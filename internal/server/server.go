package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/npclaudiu/git-server-poc/internal/config"
	"github.com/npclaudiu/git-server-poc/internal/metastore"
	"github.com/npclaudiu/git-server-poc/internal/objectstore"
)

type Server struct {
	cfg         *config.Config
	metaStore   *metastore.MetaStore
	objectStore *objectstore.ObjectStore
	httpServer  *http.Server
	wg          sync.WaitGroup
}

func New(cfg *config.Config, ms *metastore.MetaStore, os *objectstore.ObjectStore) *Server {
	s := &Server{
		cfg:         cfg,
		metaStore:   ms,
		objectStore: os,
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:    net.JoinHostPort(cfg.Server.Host, cfg.Server.Port),
		Handler: r,
	}

	return s
}

func (s *Server) Run() error {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		slog.Info("git-server-poc listening", "addr", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("error listening and serving", "err", err)
		}
	}()
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return err
	}

	s.wg.Wait()
	return nil
}

type HealthResponse struct {
	Status      string `json:"status"`
	MetaStore   string `json:"meta_store"`
	ObjectStore string `json:"object_store"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:      "ok",
		MetaStore:   "up",
		ObjectStore: "up",
	}

	errors := 0

	if err := s.metaStore.Ping(r.Context()); err != nil {
		slog.Error("metastore health check failed", "err", err)
		errors++
		resp.MetaStore = "down"
	}

	if err := s.objectStore.Ping(r.Context()); err != nil {
		slog.Error("objectstore health check failed", "err", err)
		errors++
		resp.ObjectStore = "down"
	}

	w.Header().Set("Content-Type", "application/json")
	if errors > 0 {
		resp.Status = "error"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(resp)
}
