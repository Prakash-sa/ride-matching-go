package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/example/ride-matching/internal/config"
	httpapi "github.com/example/ride-matching/internal/http"
	"github.com/example/ride-matching/internal/logging"
)

func main() {
	cfg, err := config.LoadServerConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := logging.NewLogger(cfg.LogLevel)

	if cfg.RunMigrations && cfg.PGDSN != "" {
		if err := runMigrations(cfg.PGDSN, logger); err != nil {
			logger.Error("migration failed", "error", err)
			return
		}
	}

	srv, err := httpapi.NewServer(cfg, logger)
	if err != nil {
		logger.Error("server init failed", "error", err)
		return
	}

	httpSrv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      srv,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("ride-matching listening", "addr", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server error", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	} else {
		logger.Info("server stopped cleanly")
	}
}

func runMigrations(dsn string, logger *slog.Logger) error {
	logger.Info("running migrations")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	query, err := os.ReadFile(filepath.Join("migrations", "001_create_rides.sql"))
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}

	if _, err := db.ExecContext(ctx, string(query)); err != nil {
		return fmt.Errorf("exec migration: %w", err)
	}
	logger.Info("migration applied", "file", "001_create_rides.sql")
	return nil
}
