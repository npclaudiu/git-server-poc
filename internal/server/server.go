package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"regexp"
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

	r.Get("/repositories", s.handleListRepositories)
	r.Post("/repositories", s.handleCreateRepository)
	r.Get("/repositories/{repository_id}", s.handleGetRepository)
	r.Put("/repositories/{repository_id}", s.handleUpdateRepository)
	r.Delete("/repositories/{repository_id}", s.handleDeleteRepository)

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

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Type", "application/json")

	if errors > 0 {
		resp.Status = "error"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(resp)
}

// Request/Response structs

type CreateRepositoryRequest struct {
	Name string `json:"name"`
}

type UpdateRepositoryRequest struct {
	Name string `json:"name"`
}

var validNameRegex = regexp.MustCompile(`^[a-z0-9\-_]+$`)

func isValidRepoName(name string) bool {
	return validNameRegex.MatchString(name)
}

// Handlers

func (s *Server) handleCreateRepository(w http.ResponseWriter, r *http.Request) {
	var req CreateRepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if !isValidRepoName(req.Name) {
		http.Error(w, "invalid repository name", http.StatusBadRequest)
		return
	}

	repo, err := s.metaStore.CreateRepository(r.Context(), req.Name)
	if err != nil {
		slog.Error("failed to create repository", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(repo)
}

func (s *Server) handleListRepositories(w http.ResponseWriter, r *http.Request) {
	repos, err := s.metaStore.ListRepositories(r.Context())
	if err != nil {
		slog.Error("failed to list repositories", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repos)
}

func (s *Server) handleGetRepository(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "repository_id")
	if !isValidRepoName(id) {
		http.Error(w, "invalid repository id", http.StatusBadRequest)
		return
	}

	repo, err := s.metaStore.GetRepository(r.Context(), id)
	if err != nil {
		slog.Error("failed to get repository", "id", id, "err", err)
		// TODO: handle ErrNoRows specifically
		http.Error(w, "repository not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repo)
}

func (s *Server) handleUpdateRepository(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "repository_id")
	if !isValidRepoName(id) {
		http.Error(w, "invalid repository id", http.StatusBadRequest)
		return
	}

	var req UpdateRepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if !isValidRepoName(req.Name) {
		http.Error(w, "invalid repository name", http.StatusBadRequest)
		return
	}

	repo, err := s.metaStore.UpdateRepository(r.Context(), id, req.Name)
	if err != nil {
		slog.Error("failed to update repository", "id", id, "err", err)
		// TODO: handle ErrNoRows specifically
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repo)
}

func (s *Server) handleDeleteRepository(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "repository_id")
	if !isValidRepoName(id) {
		http.Error(w, "invalid repository id", http.StatusBadRequest)
		return
	}

	if err := s.metaStore.DeleteRepository(r.Context(), id); err != nil {
		slog.Error("failed to delete repository", "id", id, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
