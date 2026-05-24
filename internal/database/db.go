// Author: Hakan Gunay
// Date: 2026-04-04
// Database connection setup, auto-creation, and migration runner

package database

import (
	"fmt"
	"time"

	"minigaraj-scraper/internal/config"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// InitDB ensures the database exists, runs migrations, and returns a connection pool.
// Steps:
//  1. Connect with admin user → check/create database
//  2. Connect with admin user to target DB → run migrations (needs DDL)
//  3. Connect with app user → return pool for application use
func InitDB(cfg config.DatabaseConfig, migrationsPath string, logger *zap.Logger) (*sqlx.DB, error) {
	// Step 1: Ensure database exists (via admin user → "postgres" database)
	if err := ensureDatabaseExists(cfg, logger); err != nil {
		return nil, fmt.Errorf("ensure database: %w", err)
	}

	// Step 2: Run migrations with admin user (needs CREATE SCHEMA, CREATE TABLE etc.)
	adminDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.InitUser, cfg.InitUserPassword, cfg.DBName, cfg.SSLMode,
	)
	adminDB, err := connectWithRetry(adminDSN, logger)
	if err != nil {
		return nil, fmt.Errorf("admin connect for migrations: %w", err)
	}
	if err := RunMigrations(adminDB, migrationsPath, logger); err != nil {
		adminDB.Close()
		return nil, err
	}
	adminDB.Close()

	// Step 3: Connect with application user for normal operations
	db, err := Connect(cfg, logger)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// ensureDatabaseExists connects to the "postgres" database with the admin user
// and creates the target database if it doesn't exist.
func ensureDatabaseExists(cfg config.DatabaseConfig, logger *zap.Logger) error {
	adminDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Host, cfg.Port, cfg.InitUser, cfg.InitUserPassword, cfg.SSLMode,
	)

	db, err := connectWithRetry(adminDSN, logger)
	if err != nil {
		return fmt.Errorf("connect to postgres db: %w", err)
	}
	defer db.Close()

	// Check if target database exists
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", cfg.DBName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check database exists: %w", err)
	}

	if exists {
		logger.Info("database already exists", zap.String("db", cfg.DBName))
		return nil
	}

	// Create database (can't use parameterized query for CREATE DATABASE)
	_, err = db.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, cfg.DBName))
	if err != nil {
		return fmt.Errorf("create database: %w", err)
	}
	logger.Info("database created", zap.String("db", cfg.DBName))

	// Ensure app user exists and has permissions
	if cfg.User != cfg.InitUser {
		if err := ensureAppUser(db, cfg, logger); err != nil {
			logger.Warn("failed to setup app user (may already exist)", zap.Error(err))
		}
	}

	return nil
}

// ensureAppUser creates the application user if it doesn't exist and grants basic permissions
func ensureAppUser(db *sqlx.DB, cfg config.DatabaseConfig, logger *zap.Logger) error {
	// Create user if not exists
	var userExists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)", cfg.User).Scan(&userExists)
	if err != nil {
		return fmt.Errorf("check user exists: %w", err)
	}

	if !userExists {
		_, err = db.Exec(fmt.Sprintf(
			`CREATE USER "%s" WITH PASSWORD '%s'`,
			cfg.User, cfg.Password,
		))
		if err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		logger.Info("app user created", zap.String("user", cfg.User))
	}

	// Grant connect permission
	_, err = db.Exec(fmt.Sprintf(`GRANT CONNECT ON DATABASE "%s" TO "%s"`, cfg.DBName, cfg.User))
	if err != nil {
		return fmt.Errorf("grant connect: %w", err)
	}

	return nil
}

// Connect creates and validates a PostgreSQL connection pool with the app user
func Connect(cfg config.DatabaseConfig, logger *zap.Logger) (*sqlx.DB, error) {
	db, err := sqlx.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.MaxLifetime)

	var pingErr error
	for i := 0; i < 10; i++ {
		pingErr = db.Ping()
		if pingErr == nil {
			break
		}
		logger.Warn("database not ready, retrying...",
			zap.Int("attempt", i+1),
			zap.Error(pingErr),
		)
		time.Sleep(2 * time.Second)
	}
	if pingErr != nil {
		return nil, fmt.Errorf("ping db after retries: %w", pingErr)
	}

	logger.Info("database connected",
		zap.String("host", cfg.Host),
		zap.String("db", cfg.DBName),
		zap.String("user", cfg.User),
	)
	return db, nil
}

// connectWithRetry opens a connection and retries ping until ready
func connectWithRetry(dsn string, logger *zap.Logger) (*sqlx.DB, error) {
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	for i := 0; i < 10; i++ {
		if err := db.Ping(); err == nil {
			return db, nil
		}
		logger.Debug("waiting for database...", zap.Int("attempt", i+1))
		time.Sleep(2 * time.Second)
	}
	db.Close()
	return nil, fmt.Errorf("database not reachable after retries")
}

// RunMigrations applies all pending migrations from the given directory
func RunMigrations(db *sqlx.DB, migrationsPath string, logger *zap.Logger) error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}

	logger.Info("migrations applied successfully")
	return nil
}
