// Author: Hakan Gunay
// Date: 2026-04-04
// Config package - reads environment variables for minigaraj-scraper

package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	App       AppConfig
	Database  DatabaseConfig
	Crawler   CrawlerConfig
	Log       LogConfig
	Schedules []ScheduleConfig
}

// ScheduleConfig holds a brand crawl schedule
type ScheduleConfig struct {
	Brand    string
	CronExpr string
}

// AppConfig holds general application settings
type AppConfig struct {
	Name   string
	Env    string // development, staging, production
	Port   int    // HTTP API port (for health/metrics)
	APIKey string // API key for authentication (empty = no auth)
}

// DatabaseConfig holds PostgreSQL connection settings (scraper DB)
type DatabaseConfig struct {
	Host             string
	Port             int
	User             string
	Password         string
	DBName           string
	SSLMode          string
	MaxOpenConns     int
	MaxIdleConns     int
	MaxLifetime      time.Duration
	InitUser         string // Admin user for DB creation (e.g. "postgres")
	InitUserPassword string // Admin user password
}

// DSN returns the PostgreSQL connection string
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// CrawlerConfig holds web crawler settings
type CrawlerConfig struct {
	// Concurrency: how many parallel requests per domain
	Parallelism int
	// Delay between requests to same domain (ms)
	RequestDelayMs int
	// Random delay added to RequestDelayMs (ms)
	RandomDelayMs int
	// Max depth to follow links
	MaxDepth int
	// Request timeout (seconds)
	TimeoutSec int
	// User agent string
	UserAgent string
	// Allowed domains (comma-separated), empty = all
	AllowedDomains string
	// Max retries per URL
	MaxRetries int
}

// LogConfig holds logging settings
type LogConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, console
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	v := viper.New()

	// App defaults
	v.SetDefault("app.name", "minigaraj-scraper")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", 8300)
	v.SetDefault("app.api_key", "")

	// Database defaults
	v.SetDefault("db.host", "localhost")
	v.SetDefault("db.port", 5433) // separate port from minigaraj-api DB
	v.SetDefault("db.user", "scraper")
	v.SetDefault("db.password", "scraper")
	v.SetDefault("db.name", "minigaraj_scraper")
	v.SetDefault("db.sslmode", "disable")
	v.SetDefault("db.max_open_conns", 10)
	v.SetDefault("db.max_idle_conns", 5)
	v.SetDefault("db.max_lifetime_minutes", 30)
	v.SetDefault("db.init_user", "postgres")
	v.SetDefault("db.init_user_password", "postgres")

	// Crawler defaults (polite but efficient)
	v.SetDefault("crawler.parallelism", 4)
	v.SetDefault("crawler.request_delay_ms", 1000)
	v.SetDefault("crawler.random_delay_ms", 500)
	v.SetDefault("crawler.max_depth", 5)
	v.SetDefault("crawler.timeout_sec", 30)
	v.SetDefault("crawler.user_agent", "MiniGarajBot/1.0 (+https://minigaraj.app/bot)")
	v.SetDefault("crawler.allowed_domains", "")
	v.SetDefault("crawler.max_retries", 3)

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")

	// Bind environment variables (SCRAPER_DB_HOST etc.)
	v.SetEnvPrefix("SCRAPER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	maxLifetime := time.Duration(v.GetInt("db.max_lifetime_minutes")) * time.Minute

	cfg := &Config{
		App: AppConfig{
			Name:   v.GetString("app.name"),
			Env:    v.GetString("app.env"),
			Port:   v.GetInt("app.port"),
			APIKey: v.GetString("app.api_key"),
		},
		Database: DatabaseConfig{
			Host:             v.GetString("db.host"),
			Port:             v.GetInt("db.port"),
			User:             v.GetString("db.user"),
			Password:         v.GetString("db.password"),
			DBName:           v.GetString("db.name"),
			SSLMode:          v.GetString("db.sslmode"),
			MaxOpenConns:     v.GetInt("db.max_open_conns"),
			MaxIdleConns:     v.GetInt("db.max_idle_conns"),
			MaxLifetime:      maxLifetime,
			InitUser:         v.GetString("db.init_user"),
			InitUserPassword: v.GetString("db.init_user_password"),
		},
		Crawler: CrawlerConfig{
			Parallelism:    v.GetInt("crawler.parallelism"),
			RequestDelayMs: v.GetInt("crawler.request_delay_ms"),
			RandomDelayMs:  v.GetInt("crawler.random_delay_ms"),
			MaxDepth:       v.GetInt("crawler.max_depth"),
			TimeoutSec:     v.GetInt("crawler.timeout_sec"),
			UserAgent:      v.GetString("crawler.user_agent"),
			AllowedDomains: v.GetString("crawler.allowed_domains"),
			MaxRetries:     v.GetInt("crawler.max_retries"),
		},
		Log: LogConfig{
			Level:  v.GetString("log.level"),
			Format: v.GetString("log.format"),
		},
		Schedules: []ScheduleConfig{
			// Default: weekly crawl for each brand (Sunday 2am, 3am, 4am)
			{Brand: "Hot Wheels", CronExpr: "0 0 2 * * 0"},
			{Brand: "Matchbox", CronExpr: "0 0 3 * * 0"},
			{Brand: "Mini GT", CronExpr: "0 0 4 * * 0"},
		},
	}

	return cfg, nil
}
