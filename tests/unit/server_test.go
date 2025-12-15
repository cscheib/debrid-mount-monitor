package unit

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chris/debrid-mount-monitor/internal/health"
	"github.com/chris/debrid-mount-monitor/internal/server"
)

// testLogger returns a silent logger for testing.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestLivenessEndpoint_HealthyMount tests liveness returns 200 for healthy mounts.
func TestLivenessEndpoint_HealthyMount(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	// Make mount healthy
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	mount.UpdateState(result, 3)

	srv := server.New([]*health.Mount{mount}, 0, testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/live", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %q", response["status"])
	}

	// Verify OpenAPI-compliant response has timestamp and mounts
	if _, ok := response["timestamp"]; !ok {
		t.Error("response missing 'timestamp' field")
	}
	if _, ok := response["mounts"]; !ok {
		t.Error("response missing 'mounts' field")
	}
}

// TestLivenessEndpoint_DegradedMount tests liveness returns 200 for degraded mounts.
// Per spec: degraded (within debounce) should NOT trigger liveness failure.
func TestLivenessEndpoint_DegradedMount(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	debounceThreshold := 3

	// Simulate degraded state (1 failure, below threshold)
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   false,
		Duration:  100 * time.Millisecond,
	}
	mount.UpdateState(result, debounceThreshold)

	if mount.GetStatus() != health.StatusDegraded {
		t.Fatalf("expected mount to be degraded, got %v", mount.GetStatus())
	}

	srv := server.New([]*health.Mount{mount}, 0, testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/live", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Degraded mounts should return 200 (only UNHEALTHY triggers 503)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 for degraded mount, got %d", rec.Code)
	}
}

// TestLivenessEndpoint_UnhealthyMount tests liveness returns 503 for unhealthy mounts.
// Per spec: liveness returns 503 when mount is UNHEALTHY (past debounce threshold).
func TestLivenessEndpoint_UnhealthyMount(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	debounceThreshold := 3

	// Simulate unhealthy state (3 failures)
	for i := 0; i < debounceThreshold; i++ {
		result := &health.CheckResult{
			Mount:     mount,
			Timestamp: time.Now(),
			Success:   false,
			Duration:  100 * time.Millisecond,
		}
		mount.UpdateState(result, debounceThreshold)
	}

	if mount.GetStatus() != health.StatusUnhealthy {
		t.Fatalf("expected mount to be unhealthy, got %v", mount.GetStatus())
	}

	srv := server.New([]*health.Mount{mount}, 0, testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/live", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Unhealthy mounts should trigger 503
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 for unhealthy mount, got %d", rec.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %q", response["status"])
	}
}

// TestLivenessEndpoint_UnknownMount tests liveness returns 200 for unknown mounts.
// Per spec: unknown (no check yet) should NOT trigger liveness failure.
func TestLivenessEndpoint_UnknownMount(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	// Mount starts in UNKNOWN state (no checks performed)

	if mount.GetStatus() != health.StatusUnknown {
		t.Fatalf("expected mount to be unknown, got %v", mount.GetStatus())
	}

	srv := server.New([]*health.Mount{mount}, 0, testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/live", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Unknown mounts should return 200 (grace period)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 for unknown mount, got %d", rec.Code)
	}
}

// TestReadinessEndpoint_AllHealthy tests readiness returns 200 when all mounts are healthy.
func TestReadinessEndpoint_AllHealthy(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	// Simulate healthy state
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	mount.UpdateState(result, 3)

	srv := server.New([]*health.Mount{mount}, 0, testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %q", response["status"])
	}
}

// TestReadinessEndpoint_UnhealthyMount tests readiness returns 503 when any mount is unhealthy.
func TestReadinessEndpoint_UnhealthyMount(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	debounceThreshold := 3

	// Simulate unhealthy state (3 failures)
	for i := 0; i < debounceThreshold; i++ {
		result := &health.CheckResult{
			Mount:     mount,
			Timestamp: time.Now(),
			Success:   false,
			Duration:  100 * time.Millisecond,
		}
		mount.UpdateState(result, debounceThreshold)
	}

	srv := server.New([]*health.Mount{mount}, 0, testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %q", response["status"])
	}
}

// TestReadinessEndpoint_DegradedMount tests readiness returns 503 for degraded mounts.
// Per spec: readiness requires HEALTHY state - degraded triggers 503.
func TestReadinessEndpoint_DegradedMount(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	debounceThreshold := 3

	// Simulate degraded state (1 failure, below threshold)
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   false,
		Duration:  100 * time.Millisecond,
	}
	mount.UpdateState(result, debounceThreshold)

	if mount.GetStatus() != health.StatusDegraded {
		t.Fatalf("expected mount to be degraded, got %v", mount.GetStatus())
	}

	srv := server.New([]*health.Mount{mount}, 0, testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Per spec: DEGRADED should return 503 for readiness
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 for degraded mount, got %d", rec.Code)
	}
}

// TestReadinessEndpoint_UnknownMount tests readiness returns 503 for unknown mounts.
// Per spec: readiness requires HEALTHY - unknown (no check yet) triggers 503.
func TestReadinessEndpoint_UnknownMount(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	// Mount starts in UNKNOWN state (no checks performed)

	if mount.GetStatus() != health.StatusUnknown {
		t.Fatalf("expected mount to be unknown, got %v", mount.GetStatus())
	}

	srv := server.New([]*health.Mount{mount}, 0, testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Per spec: UNKNOWN should return 503 for readiness
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 for unknown mount, got %d", rec.Code)
	}
}

// TestStatusEndpoint_DetailedInfo tests the status endpoint returns detailed mount info.
func TestStatusEndpoint_DetailedInfo(t *testing.T) {
	mount1 := health.NewMount("", "/mnt/test1", ".health-check", 3)
	mount2 := health.NewMount("", "/mnt/test2", ".health-check", 3)

	// Mount1: healthy
	result1 := &health.CheckResult{
		Mount:     mount1,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	mount1.UpdateState(result1, 3)

	// Mount2: also healthy
	result2 := &health.CheckResult{
		Mount:     mount2,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	mount2.UpdateState(result2, 3)

	srv := server.New([]*health.Mount{mount1, mount2}, 0, testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response struct {
		Status    string `json:"status"`
		Timestamp string `json:"timestamp"`
		Mounts    []struct {
			Path         string `json:"path"`
			Status       string `json:"status"`
			FailureCount int    `json:"failure_count"`
		} `json:"mounts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("expected overall status 'healthy', got %q", response.Status)
	}

	if response.Timestamp == "" {
		t.Error("response missing timestamp")
	}

	if len(response.Mounts) != 2 {
		t.Fatalf("expected 2 mounts, got %d", len(response.Mounts))
	}

	if response.Mounts[0].Status != "healthy" {
		t.Errorf("expected mount1 status 'healthy', got %q", response.Mounts[0].Status)
	}
	if response.Mounts[1].Status != "healthy" {
		t.Errorf("expected mount2 status 'healthy', got %q", response.Mounts[1].Status)
	}
}

// TestStatusEndpoint_DegradedMount tests status returns 503 when any mount is degraded.
// Per spec: status endpoint uses same logic as readiness.
func TestStatusEndpoint_DegradedMount(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)

	// Simulate degraded state (1 failure)
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   false,
		Duration:  100 * time.Millisecond,
	}
	mount.UpdateState(result, 3)

	srv := server.New([]*health.Mount{mount}, 0, testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Per spec: degraded should return 503
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 for degraded mount, got %d", rec.Code)
	}
}

// TestEndpoints_MethodNotAllowed tests that non-GET methods return 405.
func TestEndpoints_MethodNotAllowed(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	srv := server.New([]*health.Mount{mount}, 0, testLogger())
	handler := createServerHandler(srv)

	endpoints := []string{"/healthz/live", "/healthz/ready", "/healthz/status"}
	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, endpoint := range endpoints {
		for _, method := range methods {
			t.Run(method+"_"+endpoint, func(t *testing.T) {
				req := httptest.NewRequest(method, endpoint, nil)
				rec := httptest.NewRecorder()

				handler.ServeHTTP(rec, req)

				if rec.Code != http.StatusMethodNotAllowed {
					t.Errorf("expected status 405 for %s %s, got %d", method, endpoint, rec.Code)
				}
			})
		}
	}
}

// createServerHandler creates an HTTP handler from the real server for testing.
// Uses the server's Handler() method to get the internal mux for direct testing.
func createServerHandler(srv *server.Server) http.Handler {
	return srv.Handler()
}
