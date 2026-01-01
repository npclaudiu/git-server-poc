package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/npclaudiu/git-server-poc/internal/config"
	"github.com/npclaudiu/git-server-poc/internal/metastore"
	"github.com/npclaudiu/git-server-poc/internal/objectstore"
	"github.com/npclaudiu/git-server-poc/internal/server"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("failed to load config", "err", err)
		os.Exit(1)
	}

	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(cfg.Log.Level)); err != nil {
		logLevel = slog.LevelInfo
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Warn("invalid log level in config, defaulting to info", "level", cfg.Log.Level, "err", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	metaStore, err := metastore.New(ctx, metastore.Options{
		Host:     cfg.MetaStore.Host,
		Port:     cfg.MetaStore.Port,
		User:     cfg.MetaStore.User,
		Password: cfg.MetaStore.Password,
		DBName:   cfg.MetaStore.DBName,
		SSLMode:  cfg.MetaStore.SSLMode,
	})
	if err != nil {
		slog.Error("failed to create metastore", "err", err)
		os.Exit(1)
	}
	defer metaStore.Close()

	objStore, err := objectstore.New(ctx, objectstore.Options{
		Endpoint:  cfg.ObjectStore.Endpoint,
		AccessKey: cfg.ObjectStore.AccessKeyID,
		SecretKey: cfg.ObjectStore.SecretAccessKey,
		Bucket:    cfg.ObjectStore.Bucket,
		Region:    cfg.ObjectStore.Region,
	})
	if err != nil {
		slog.Error("failed to create object store", "err", err)
		os.Exit(1)
	}

	if err := objStore.EnsureBucket(ctx); err != nil {
		slog.Error("failed to ensure bucket exists", "err", err)
		os.Exit(1)
	}

	srv := server.New(cfg, metaStore, objStore)

	if err := srv.Run(); err != nil {
		slog.Error("failed to run server", "err", err)
		os.Exit(1)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("failed to shutdown server", "err", err)
		os.Exit(1)
	}
}
