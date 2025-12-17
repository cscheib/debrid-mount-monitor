package unit

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cscheib/debrid-mount-monitor/internal/health"
	"github.com/cscheib/debrid-mount-monitor/internal/server"
	"github.com/matryer/is"
)

// testLogger returns a silent logger for testing.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestLivenessEndpoint_HealthyMount tests liveness returns 200 for healthy mounts.
func TestLivenessEndpoint_HealthyMount(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	// Make mount healthy
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	mount.UpdateState(result, 3)

	srv := server.New([]*health.Mount{mount}, 0, "test", testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/live", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK) // status should be 200

	var response map[string]any
	is.NoErr(json.Unmarshal(rec.Body.Bytes(), &response)) // should parse response

	is.Equal(response["status"], "healthy") // status should be healthy
	is.True(response["timestamp"] != nil)   // should have timestamp
	is.True(response["mounts"] != nil)      // should have mounts
}

// TestLivenessEndpoint_DegradedMount tests liveness returns 200 for degraded mounts.
// Per spec: degraded (within failure threshold) should NOT trigger liveness failure.
func TestLivenessEndpoint_DegradedMount(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	failureThreshold := 3

	// Simulate degraded state (1 failure, below threshold)
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   false,
		Duration:  100 * time.Millisecond,
	}
	mount.UpdateState(result, failureThreshold)

	is.Equal(mount.GetStatus(), health.StatusDegraded) // mount should be degraded

	srv := server.New([]*health.Mount{mount}, 0, "test", testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/live", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Degraded mounts should return 200 (only UNHEALTHY triggers 503)
	is.Equal(rec.Code, http.StatusOK) // degraded mount should return 200
}

// TestLivenessEndpoint_UnhealthyMount tests liveness returns 503 for unhealthy mounts.
// Per spec: liveness returns 503 when mount is UNHEALTHY (past failure threshold).
func TestLivenessEndpoint_UnhealthyMount(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	failureThreshold := 3

	// Simulate unhealthy state (3 failures)
	for i := 0; i < failureThreshold; i++ {
		result := &health.CheckResult{
			Mount:     mount,
			Timestamp: time.Now(),
			Success:   false,
			Duration:  100 * time.Millisecond,
		}
		mount.UpdateState(result, failureThreshold)
	}

	is.Equal(mount.GetStatus(), health.StatusUnhealthy) // mount should be unhealthy

	srv := server.New([]*health.Mount{mount}, 0, "test", testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/live", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Unhealthy mounts should trigger 503
	is.Equal(rec.Code, http.StatusServiceUnavailable) // unhealthy mount should return 503

	var response map[string]any
	is.NoErr(json.Unmarshal(rec.Body.Bytes(), &response)) // should parse response

	is.Equal(response["status"], "unhealthy") // status should be unhealthy
}

// TestLivenessEndpoint_UnknownMount tests liveness returns 200 for unknown mounts.
// Per spec: unknown (no check yet) should NOT trigger liveness failure.
func TestLivenessEndpoint_UnknownMount(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	// Mount starts in UNKNOWN state (no checks performed)

	is.Equal(mount.GetStatus(), health.StatusUnknown) // mount should be unknown

	srv := server.New([]*health.Mount{mount}, 0, "test", testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/live", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Unknown mounts should return 200 (grace period)
	is.Equal(rec.Code, http.StatusOK) // unknown mount should return 200
}

// TestReadinessEndpoint_AllHealthy tests readiness returns 200 when all mounts are healthy.
func TestReadinessEndpoint_AllHealthy(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	// Simulate healthy state
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	mount.UpdateState(result, 3)

	srv := server.New([]*health.Mount{mount}, 0, "test", testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK) // status should be 200

	var response map[string]any
	is.NoErr(json.Unmarshal(rec.Body.Bytes(), &response)) // should parse response

	is.Equal(response["status"], "healthy") // status should be healthy
}

// TestReadinessEndpoint_UnhealthyMount tests readiness returns 503 when any mount is unhealthy.
func TestReadinessEndpoint_UnhealthyMount(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	failureThreshold := 3

	// Simulate unhealthy state (3 failures)
	for i := 0; i < failureThreshold; i++ {
		result := &health.CheckResult{
			Mount:     mount,
			Timestamp: time.Now(),
			Success:   false,
			Duration:  100 * time.Millisecond,
		}
		mount.UpdateState(result, failureThreshold)
	}

	srv := server.New([]*health.Mount{mount}, 0, "test", testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusServiceUnavailable) // status should be 503

	var response map[string]any
	is.NoErr(json.Unmarshal(rec.Body.Bytes(), &response)) // should parse response

	is.Equal(response["status"], "unhealthy") // status should be unhealthy
}

// TestReadinessEndpoint_DegradedMount tests readiness returns 503 for degraded mounts.
// Per spec: readiness requires HEALTHY state - degraded triggers 503.
func TestReadinessEndpoint_DegradedMount(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	failureThreshold := 3

	// Simulate degraded state (1 failure, below threshold)
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   false,
		Duration:  100 * time.Millisecond,
	}
	mount.UpdateState(result, failureThreshold)

	is.Equal(mount.GetStatus(), health.StatusDegraded) // mount should be degraded

	srv := server.New([]*health.Mount{mount}, 0, "test", testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Per spec: DEGRADED should return 503 for readiness
	is.Equal(rec.Code, http.StatusServiceUnavailable) // degraded mount should return 503
}

// TestReadinessEndpoint_UnknownMount tests readiness returns 503 for unknown mounts.
// Per spec: readiness requires HEALTHY - unknown (no check yet) triggers 503.
func TestReadinessEndpoint_UnknownMount(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	// Mount starts in UNKNOWN state (no checks performed)

	is.Equal(mount.GetStatus(), health.StatusUnknown) // mount should be unknown

	srv := server.New([]*health.Mount{mount}, 0, "test", testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Per spec: UNKNOWN should return 503 for readiness
	is.Equal(rec.Code, http.StatusServiceUnavailable) // unknown mount should return 503
}

// TestStatusEndpoint_DetailedInfo tests the status endpoint returns detailed mount info.
func TestStatusEndpoint_DetailedInfo(t *testing.T) {
	is := is.New(t)

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

	srv := server.New([]*health.Mount{mount1, mount2}, 0, "test", testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK) // status should be 200

	var response struct {
		Status    string `json:"status"`
		Timestamp string `json:"timestamp"`
		Mounts    []struct {
			Path         string `json:"path"`
			Status       string `json:"status"`
			FailureCount int    `json:"failure_count"`
		} `json:"mounts"`
	}
	is.NoErr(json.Unmarshal(rec.Body.Bytes(), &response)) // should parse response

	is.Equal(response.Status, "healthy")           // overall status
	is.True(response.Timestamp != "")              // should have timestamp
	is.Equal(len(response.Mounts), 2)              // should have 2 mounts
	is.Equal(response.Mounts[0].Status, "healthy") // mount1 status
	is.Equal(response.Mounts[1].Status, "healthy") // mount2 status
}

// TestStatusEndpoint_DegradedMount tests status returns 503 when any mount is degraded.
// Per spec: status endpoint uses same logic as readiness.
func TestStatusEndpoint_DegradedMount(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)

	// Simulate degraded state (1 failure)
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   false,
		Duration:  100 * time.Millisecond,
	}
	mount.UpdateState(result, 3)

	srv := server.New([]*health.Mount{mount}, 0, "test", testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Per spec: degraded should return 503
	is.Equal(rec.Code, http.StatusServiceUnavailable) // degraded should return 503
}

// TestStatusEndpoint_IncludesMountName tests that the status endpoint includes mount name when configured.
func TestStatusEndpoint_IncludesMountName(t *testing.T) {
	is := is.New(t)

	// Create mount with a name
	mountWithName := health.NewMount("debrid-movies", "/mnt/movies", ".health-check", 3)
	// Create mount without a name
	mountWithoutName := health.NewMount("", "/mnt/tv", ".health-check", 3)

	// Make both mounts healthy
	result1 := &health.CheckResult{
		Mount:     mountWithName,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	mountWithName.UpdateState(result1, 3)

	result2 := &health.CheckResult{
		Mount:     mountWithoutName,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	mountWithoutName.UpdateState(result2, 3)

	srv := server.New([]*health.Mount{mountWithName, mountWithoutName}, 0, "test", testLogger())
	handler := createServerHandler(srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK) // status should be 200

	var response struct {
		Status string `json:"status"`
		Mounts []struct {
			Name   string `json:"name"`
			Path   string `json:"path"`
			Status string `json:"status"`
		} `json:"mounts"`
	}
	is.NoErr(json.Unmarshal(rec.Body.Bytes(), &response)) // should parse response

	is.Equal(len(response.Mounts), 2)                  // should have 2 mounts
	is.Equal(response.Mounts[0].Name, "debrid-movies") // mount with name
	is.Equal(response.Mounts[0].Path, "/mnt/movies")   // mount path
	is.Equal(response.Mounts[1].Name, "")              // mount without name
	is.Equal(response.Mounts[1].Path, "/mnt/tv")       // mount path
}

// TestEndpoints_MethodNotAllowed tests that non-GET methods return 405.
func TestEndpoints_MethodNotAllowed(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	srv := server.New([]*health.Mount{mount}, 0, "test", testLogger())
	handler := createServerHandler(srv)

	endpoints := []string{"/healthz/live", "/healthz/ready", "/healthz/status"}
	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, endpoint := range endpoints {
		for _, method := range methods {
			t.Run(method+"_"+endpoint, func(t *testing.T) {
				is := is.New(t)

				req := httptest.NewRequest(method, endpoint, nil)
				rec := httptest.NewRecorder()

				handler.ServeHTTP(rec, req)

				is.Equal(rec.Code, http.StatusMethodNotAllowed) // should return 405
			})
		}
	}
}

// createServerHandler creates an HTTP handler from the real server for testing.
// Uses the server's Handler() method to get the internal mux for direct testing.
func createServerHandler(srv *server.Server) http.Handler {
	return srv.Handler()
}
