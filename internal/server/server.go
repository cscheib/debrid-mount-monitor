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

// Handler returns the HTTP handler for testing purposes.
func (s *Server) Handler() http.Handler {
	return s.server.Handler
}

// handleLiveness responds to liveness probe requests.
// Returns 200 OK if no mount is UNHEALTHY (past debounce threshold).
// Per spec: HEALTHY, DEGRADED, and UNKNOWN states return 200; only UNHEALTHY returns 503.
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if any mount is confirmed unhealthy (past debounce threshold)
	allAlive := true
	for _, mount := range s.mounts {
		status := mount.GetStatus()
		// Only UNHEALTHY (past debounce) triggers liveness failure
		if status == health.StatusUnhealthy {
			allAlive = false
			break
		}
	}

	response := s.buildProbeResponse(allAlive)
	w.Header().Set("Content-Type", "application/json")

	if allAlive {
		s.logger.Debug("probe request", "endpoint", "/healthz/live", "status", http.StatusOK, "result", "alive")
		w.WriteHeader(http.StatusOK)
	} else {
		s.logger.Warn("probe request", "endpoint", "/healthz/live", "status", http.StatusServiceUnavailable, "result", "unhealthy")
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("failed to encode liveness response", "error", err)
	}
}

// handleReadiness responds to readiness probe requests.
// Returns 200 OK only if ALL mounts are HEALTHY, 503 Service Unavailable otherwise.
// Per spec: DEGRADED, UNHEALTHY, and UNKNOWN states all return 503.
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	allHealthy := true
	for _, mount := range s.mounts {
		status := mount.GetStatus()
		// Only HEALTHY state is considered ready - DEGRADED, UNHEALTHY, and UNKNOWN all fail
		if status != health.StatusHealthy {
			allHealthy = false
			break
		}
	}

	response := s.buildProbeResponse(allHealthy)
	w.Header().Set("Content-Type", "application/json")

	if allHealthy {
		s.logger.Debug("probe request", "endpoint", "/healthz/ready", "status", http.StatusOK, "result", "ready")
		w.WriteHeader(http.StatusOK)
	} else {
		s.logger.Info("probe request", "endpoint", "/healthz/ready", "status", http.StatusServiceUnavailable, "result", "not_ready")
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("failed to encode readiness response", "error", err)
	}
}

// MountStatusResponse represents the status of a single mount.
type MountStatusResponse struct {
	Name         string `json:"name,omitempty"`
	Path         string `json:"path"`
	Status       string `json:"status"`
	LastCheck    string `json:"last_check,omitempty"`
	FailureCount int    `json:"failure_count"`
	LastError    string `json:"last_error,omitempty"`
}

// StatusResponse represents the overall status response.
// This matches the OpenAPI ProbeResponse schema.
type StatusResponse struct {
	Status    string                `json:"status"`
	Timestamp string                `json:"timestamp"`
	Mounts    []MountStatusResponse `json:"mounts"`
}

// buildProbeResponse creates a response that matches the OpenAPI ProbeResponse schema.
func (s *Server) buildProbeResponse(isHealthy bool) StatusResponse {
	status := "healthy"
	if !isHealthy {
		status = "unhealthy"
	}

	mountStatuses := make([]MountStatusResponse, len(s.mounts))
	for i, mount := range s.mounts {
		snapshot := mount.Snapshot()

		lastCheck := ""
		if !snapshot.LastCheck.IsZero() {
			lastCheck = snapshot.LastCheck.Format(time.RFC3339)
		}

		mountStatuses[i] = MountStatusResponse{
			Name:         snapshot.Name,
			Path:         snapshot.Path,
			Status:       snapshot.Status.String(),
			LastCheck:    lastCheck,
			FailureCount: snapshot.FailureCount,
			LastError:    snapshot.LastError,
		}
	}

	return StatusResponse{
		Status:    status,
		Timestamp: time.Now().Format(time.RFC3339),
		Mounts:    mountStatuses,
	}
}

// handleStatus responds with detailed status of all mounts.
// Uses the same logic as readiness: any non-HEALTHY mount results in 503.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if all mounts are healthy (same logic as readiness)
	overallHealthy := true
	for _, mount := range s.mounts {
		if mount.GetStatus() != health.StatusHealthy {
			overallHealthy = false
			break
		}
	}

	response := s.buildProbeResponse(overallHealthy)
	w.Header().Set("Content-Type", "application/json")

	if overallHealthy {
		s.logger.Debug("probe request", "endpoint", "/healthz/status", "status", http.StatusOK, "result", response.Status)
		w.WriteHeader(http.StatusOK)
	} else {
		s.logger.Info("probe request", "endpoint", "/healthz/status", "status", http.StatusServiceUnavailable, "result", response.Status)
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("failed to encode status response", "error", err)
	}
}
