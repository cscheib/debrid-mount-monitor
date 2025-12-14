package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chris/debrid-mount-monitor/internal/health"
)

func TestLivenessEndpoint_AlwaysReturns200(t *testing.T) {
	mount := health.NewMount("/mnt/test", ".health-check")
	handler := createTestHandler([]*health.Mount{mount})

	req := httptest.NewRequest(http.MethodGet, "/healthz/live", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "alive" {
		t.Errorf("expected status 'alive', got %q", response["status"])
	}
}

func TestReadinessEndpoint_AllHealthy(t *testing.T) {
	mount := health.NewMount("/mnt/test", ".health-check")
	// Simulate healthy state
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	mount.UpdateState(result, 3)

	handler := createTestHandler([]*health.Mount{mount})

	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "ready" {
		t.Errorf("expected status 'ready', got %q", response["status"])
	}
}

func TestReadinessEndpoint_UnhealthyMount(t *testing.T) {
	mount := health.NewMount("/mnt/test", ".health-check")
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

	handler := createTestHandler([]*health.Mount{mount})

	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "not_ready" {
		t.Errorf("expected status 'not_ready', got %q", response["status"])
	}
}

func TestReadinessEndpoint_DegradedMount_StillReady(t *testing.T) {
	mount := health.NewMount("/mnt/test", ".health-check")
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

	handler := createTestHandler([]*health.Mount{mount})

	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Degraded mounts should still return 200 (only unhealthy triggers 503)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 for degraded mount, got %d", rec.Code)
	}
}

func TestStatusEndpoint_DetailedInfo(t *testing.T) {
	mount1 := health.NewMount("/mnt/test1", ".health-check")
	mount2 := health.NewMount("/mnt/test2", ".health-check")

	// Mount1: healthy
	result1 := &health.CheckResult{
		Mount:     mount1,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	mount1.UpdateState(result1, 3)

	// Mount2: degraded (1 failure)
	result2 := &health.CheckResult{
		Mount:     mount2,
		Timestamp: time.Now(),
		Success:   false,
		Duration:  100 * time.Millisecond,
	}
	mount2.UpdateState(result2, 3)

	handler := createTestHandler([]*health.Mount{mount1, mount2})

	req := httptest.NewRequest(http.MethodGet, "/healthz/status", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response struct {
		Status string `json:"status"`
		Mounts []struct {
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

	if len(response.Mounts) != 2 {
		t.Fatalf("expected 2 mounts, got %d", len(response.Mounts))
	}

	if response.Mounts[0].Status != "healthy" {
		t.Errorf("expected mount1 status 'healthy', got %q", response.Mounts[0].Status)
	}
	if response.Mounts[1].Status != "degraded" {
		t.Errorf("expected mount2 status 'degraded', got %q", response.Mounts[1].Status)
	}
}

func TestEndpoints_MethodNotAllowed(t *testing.T) {
	mount := health.NewMount("/mnt/test", ".health-check")
	handler := createTestHandler([]*health.Mount{mount})

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

// createTestHandler creates an HTTP handler for testing without starting a server.
func createTestHandler(mounts []*health.Mount) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz/live", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	})

	mux.HandleFunc("/healthz/ready", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		allHealthy := true
		for _, mount := range mounts {
			if mount.GetStatus() == health.StatusUnhealthy {
				allHealthy = false
				break
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if allHealthy {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "not_ready"})
		}
	})

	mux.HandleFunc("/healthz/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		overallHealthy := true
		mountStatuses := make([]map[string]any, len(mounts))

		for i, mount := range mounts {
			snapshot := mount.Snapshot()
			if snapshot.Status == health.StatusUnhealthy {
				overallHealthy = false
			}

			mountStatuses[i] = map[string]any{
				"path":          snapshot.Path,
				"status":        snapshot.Status.String(),
				"failure_count": snapshot.FailureCount,
			}
			if snapshot.LastError != "" {
				mountStatuses[i]["last_error"] = snapshot.LastError
			}
		}

		status := "healthy"
		if !overallHealthy {
			status = "unhealthy"
		}

		w.Header().Set("Content-Type", "application/json")
		if overallHealthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"status": status,
			"mounts": mountStatuses,
		})
	})

	return mux
}
