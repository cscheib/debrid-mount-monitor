// Package server provides the HTTP server for health probe endpoints.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/chris/debrid-mount-monitor/internal/health"
)

// Server provides HTTP endpoints for health probes.
type Server struct {
	mounts []*health.Mount
	port   int
	logger *slog.Logger
	server *http.Server
	mu     sync.RWMutex
}

// New creates a new Server instance.
func New(mounts []*health.Mount, port int, logger *slog.Logger) *Server {
	s := &Server{
		mounts: mounts,
		port:   port,
		logger: logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz/live", s.handleLiveness)
	mux.HandleFunc("/healthz/ready", s.handleReadiness)
	mux.HandleFunc("/healthz/status", s.handleStatus)

	s.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return s
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("http server error", "error", err)
		}
	}()
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// handleLiveness responds to liveness probe requests.
// Returns 200 OK if the service is running (always alive unless server is down).
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "alive"}); err != nil {
		s.logger.Error("failed to encode liveness response", "error", err)
	}
}

// handleReadiness responds to readiness probe requests.
// Returns 200 OK if all mounts are healthy, 503 Service Unavailable otherwise.
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	allHealthy := true
	for _, mount := range s.mounts {
		status := mount.GetStatus()
		if status == health.StatusUnhealthy {
			allHealthy = false
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if allHealthy {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ready"}); err != nil {
			s.logger.Error("failed to encode readiness response", "error", err)
		}
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "not_ready"}); err != nil {
			s.logger.Error("failed to encode readiness response", "error", err)
		}
	}
}

// MountStatusResponse represents the status of a single mount.
type MountStatusResponse struct {
	Path         string `json:"path"`
	Status       string `json:"status"`
	LastCheck    string `json:"last_check,omitempty"`
	FailureCount int    `json:"failure_count"`
	LastError    string `json:"last_error,omitempty"`
}

// StatusResponse represents the overall status response.
type StatusResponse struct {
	Status string                `json:"status"`
	Mounts []MountStatusResponse `json:"mounts"`
}

// handleStatus responds with detailed status of all mounts.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	overallHealthy := true
	mountStatuses := make([]MountStatusResponse, len(s.mounts))

	for i, mount := range s.mounts {
		snapshot := mount.Snapshot()

		if snapshot.Status == health.StatusUnhealthy {
			overallHealthy = false
		}

		lastCheck := ""
		if !snapshot.LastCheck.IsZero() {
			lastCheck = snapshot.LastCheck.Format(time.RFC3339)
		}

		mountStatuses[i] = MountStatusResponse{
			Path:         snapshot.Path,
			Status:       snapshot.Status.String(),
			LastCheck:    lastCheck,
			FailureCount: snapshot.FailureCount,
			LastError:    snapshot.LastError,
		}
	}

	overallStatus := "healthy"
	if !overallHealthy {
		overallStatus = "unhealthy"
	}

	response := StatusResponse{
		Status: overallStatus,
		Mounts: mountStatuses,
	}

	w.Header().Set("Content-Type", "application/json")
	if overallHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("failed to encode status response", "error", err)
	}
}
