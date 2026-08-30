package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/alexedwards/scs/v2"
	_ "modernc.org/sqlite"

	"github.com/norrs/start/db/migrations"
	"github.com/norrs/start/internal/auth"
	"github.com/norrs/start/internal/catalog"
	"github.com/norrs/start/internal/config"
	"github.com/norrs/start/internal/seed"
	"github.com/norrs/start/internal/server"
	sessionstore "github.com/norrs/start/internal/session"
)

var version = "dev"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		if err := healthcheck(); err != nil {
			slog.Error("healthcheck failed", "error", err)
			os.Exit(1)
		}
		return
	}
	if err := run(); err != nil {
		slog.Error("start failed", "error", err)
		os.Exit(1)
	}
}

func healthcheck() error {
	url := os.Getenv("STARTPAGE_HEALTHCHECK_URL")
	if url == "" {
		url = "http://127.0.0.1:8080/healthz"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", response.Status)
	}
	return nil
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	configureLogging(cfg.LogLevel)
	if dir := filepath.Dir(cfg.DatabasePath); dir != "." {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("create database directory: %w", err)
		}
	}
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", cfg.DatabasePath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("open SQLite: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	if err := migrations.Up(db); err != nil {
		return err
	}
	cat := catalog.New(db)
	if info, statErr := os.Stat(cfg.SeedPath); statErr == nil && !info.IsDir() {
		doc, loadErr := seed.Load(cfg.SeedPath)
		if loadErr != nil {
			return loadErr
		}
		if err := cat.ImportSeedIfEmpty(context.Background(), doc); err != nil {
			return fmt.Errorf("import seed: %w", err)
		}
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	sessions := scs.New()
	sessions.Store = sessionstore.NewSQLiteStore(db)
	sessions.Lifetime = cfg.SessionLifetime
	sessions.IdleTimeout = cfg.SessionIdleTimeout
	sessions.Cookie.Name = "start_session"
	sessions.Cookie.HttpOnly = true
	sessions.Cookie.Secure = !cfg.DevMode
	sessions.Cookie.SameSite = http.SameSiteLaxMode
	sessions.Cookie.Persist = true
	authn, err := auth.New(context.Background(), cfg, cat, sessions)
	if err != nil {
		return err
	}
	app, err := server.New(cfg, db, cat, authn, sessions)
	if err != nil {
		return err
	}
	httpServer := &http.Server{Addr: cfg.ListenAddr, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 90 * time.Second}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("Start listening", "address", cfg.ListenAddr, "base_url", cfg.BaseURL, "version", version)
		errCh <- httpServer.ListenAndServe()
	}()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func configureLogging(level string) {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l})))
}
