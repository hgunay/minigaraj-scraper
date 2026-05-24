// Author: Hakan Gunay
// Date: 2026-04-04
// MiniGaraj Scraper - main entrypoint

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"minigaraj-scraper/internal/api"
	"minigaraj-scraper/internal/config"
	"minigaraj-scraper/internal/crawler/manager"
	"minigaraj-scraper/internal/database"
	"minigaraj-scraper/internal/scheduler"
	"minigaraj-scraper/internal/storage"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load error: %v\n", err)
		os.Exit(1)
	}

	// Build logger
	logger := buildLogger(cfg.Log)
	defer logger.Sync()

	logger.Info("minigaraj-scraper starting",
		zap.String("env", cfg.App.Env),
		zap.Int("port", cfg.App.Port),
	)

	// Initialize database (auto-create + migrations + connect)
	db, err := database.InitDB(cfg.Database, "./migrations", logger)
	if err != nil {
		logger.Fatal("database initialization failed", zap.Error(err))
	}
	defer db.Close()

	// Build dependencies
	repo := storage.New(db, logger)
	mgr := manager.New(cfg.Crawler, repo, logger)
	handler := api.New(mgr, repo, logger, cfg.App.APIKey)

	// HTTP server
	mux := http.NewServeMux()
	appHandler := handler.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      corsMiddleware(appHandler),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start cron scheduler
	sched := scheduler.New(mgr, logger)
	for _, s := range cfg.Schedules {
		if err := sched.AddSchedule(scheduler.Schedule{
			Brand:    s.Brand,
			CronExpr: s.CronExpr,
		}); err != nil {
			logger.Error("failed to add schedule",
				zap.String("brand", s.Brand),
				zap.Error(err),
			)
		}
	}
	sched.Start()

	// Start server
	go func() {
		logger.Info("HTTP server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")

	sched.Stop()
	mgr.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}
	logger.Info("shutdown complete")
}

// corsMiddleware adds CORS headers for minigaraj-admin access
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func buildLogger(cfg config.LogConfig) *zap.Logger {
	level := zap.InfoLevel
	switch cfg.Level {
	case "debug":
		level = zap.DebugLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	}

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "time"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	zapCfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      cfg.Level == "debug",
		Encoding:         cfg.Format,
		EncoderConfig:    encCfg,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	if cfg.Format == "console" {
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, _ := zapCfg.Build()
	return logger
}
