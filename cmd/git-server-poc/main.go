package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/npclaudiu/git-server-poc/internal/config"
	"github.com/npclaudiu/git-server-poc/internal/metastore"
)

func run(ctx context.Context, cfg *config.Config, ms *metastore.Metastore) error {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	httpServer := &http.Server{
		Addr:    net.JoinHostPort(cfg.Server.Host, cfg.Server.Port),
		Handler: r,
	}

	go func() {
		slog.Info("git-server-poc listening", "addr", httpServer.Addr)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("error listening and serving", "err", err)
		}
	}()

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()
		<-ctx.Done()

		slog.Info("shutting down server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("failed to shut down git-server-poc", "err", err)
		}

		slog.Info("closing database connection")
		ms.Close()
	}()

	wg.Wait()

	return nil
}

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("failed to load config", "err", err)
		os.Exit(1)
	}

	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(cfg.Log.Level)); err != nil {
		lvl = slog.LevelInfo
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Warn("invalid log level in config, defaulting to info", "level", cfg.Log.Level, "err", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
	slog.SetDefault(logger)

	ms, err := metastore.New(ctx, cfg.Database)
	if err != nil {
		slog.Error("failed to create metastore", "err", err)
		os.Exit(1)
	}
	defer ms.Close()

	if err := run(ctx, cfg, ms); err != nil {
		slog.Error("runtime error", "err", err)
		os.Exit(1)
	}
}
