package unit

import (
	"errors"
	"testing"
	"time"

	"github.com/chris/debrid-mount-monitor/internal/health"
)

func TestNewMount(t *testing.T) {
	mount := health.NewMount("test-mount", "/mnt/test", ".health-check", 3)

	if mount.Name != "test-mount" {
		t.Errorf("expected name 'test-mount', got %q", mount.Name)
	}
	if mount.Path != "/mnt/test" {
		t.Errorf("expected path '/mnt/test', got %q", mount.Path)
	}
	if mount.CanaryPath != "/mnt/test/.health-check" {
		t.Errorf("expected canary path '/mnt/test/.health-check', got %q", mount.CanaryPath)
	}
	if mount.FailureThreshold != 3 {
		t.Errorf("expected failure threshold 3, got %d", mount.FailureThreshold)
	}
	if mount.GetStatus() != health.StatusUnknown {
		t.Errorf("expected initial status Unknown, got %v", mount.GetStatus())
	}
	if mount.GetFailureCount() != 0 {
		t.Errorf("expected initial failure count 0, got %d", mount.GetFailureCount())
	}
}

func TestNewMount_TrailingSlash(t *testing.T) {
	mount := health.NewMount("", "/mnt/test/", ".health-check", 3)

	if mount.CanaryPath != "/mnt/test/.health-check" {
		t.Errorf("expected canary path '/mnt/test/.health-check', got %q", mount.CanaryPath)
	}
}

func TestHealthStatus_String(t *testing.T) {
	tests := []struct {
		status   health.HealthStatus
		expected string
	}{
		{health.StatusUnknown, "unknown"},
		{health.StatusHealthy, "healthy"},
		{health.StatusDegraded, "degraded"},
		{health.StatusUnhealthy, "unhealthy"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestMount_UpdateState_SuccessfulCheck(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	debounceThreshold := 3

	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
		Error:     nil,
	}

	transition := mount.UpdateState(result, debounceThreshold)

	if mount.GetStatus() != health.StatusHealthy {
		t.Errorf("expected status Healthy, got %v", mount.GetStatus())
	}
	if mount.GetFailureCount() != 0 {
		t.Errorf("expected failure count 0, got %d", mount.GetFailureCount())
	}
	if transition == nil {
		t.Error("expected state transition from Unknown to Healthy")
	}
	if transition != nil && transition.NewState != health.StatusHealthy {
		t.Errorf("expected transition to Healthy, got %v", transition.NewState)
	}
}

func TestMount_UpdateState_FailedCheck_Degraded(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	debounceThreshold := 3

	// First failure - should go to Degraded
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   false,
		Duration:  100 * time.Millisecond,
		Error:     errors.New("read timeout"),
	}

	transition := mount.UpdateState(result, debounceThreshold)

	if mount.GetStatus() != health.StatusDegraded {
		t.Errorf("expected status Degraded, got %v", mount.GetStatus())
	}
	if mount.GetFailureCount() != 1 {
		t.Errorf("expected failure count 1, got %d", mount.GetFailureCount())
	}
	if transition == nil {
		t.Error("expected state transition")
	}
}

func TestMount_UpdateState_FailedCheck_Unhealthy(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	debounceThreshold := 3

	// Simulate 3 consecutive failures
	for i := 0; i < debounceThreshold; i++ {
		result := &health.CheckResult{
			Mount:     mount,
			Timestamp: time.Now(),
			Success:   false,
			Duration:  100 * time.Millisecond,
			Error:     errors.New("read timeout"),
		}
		mount.UpdateState(result, debounceThreshold)
	}

	if mount.GetStatus() != health.StatusUnhealthy {
		t.Errorf("expected status Unhealthy after %d failures, got %v", debounceThreshold, mount.GetStatus())
	}
	if mount.GetFailureCount() != debounceThreshold {
		t.Errorf("expected failure count %d, got %d", debounceThreshold, mount.GetFailureCount())
	}
}

func TestMount_UpdateState_RecoveryFromUnhealthy(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	debounceThreshold := 3

	// Put mount into unhealthy state
	for i := 0; i < debounceThreshold; i++ {
		result := &health.CheckResult{
			Mount:     mount,
			Timestamp: time.Now(),
			Success:   false,
			Duration:  100 * time.Millisecond,
			Error:     errors.New("read timeout"),
		}
		mount.UpdateState(result, debounceThreshold)
	}

	if mount.GetStatus() != health.StatusUnhealthy {
		t.Fatalf("expected mount to be Unhealthy, got %v", mount.GetStatus())
	}

	// Recovery with successful check
	successResult := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
		Error:     nil,
	}

	transition := mount.UpdateState(successResult, debounceThreshold)

	if mount.GetStatus() != health.StatusHealthy {
		t.Errorf("expected status Healthy after recovery, got %v", mount.GetStatus())
	}
	if mount.GetFailureCount() != 0 {
		t.Errorf("expected failure count 0 after recovery, got %d", mount.GetFailureCount())
	}
	if transition == nil {
		t.Error("expected recovery transition")
	}
	if transition != nil && transition.Trigger != "recovered" {
		t.Errorf("expected trigger 'recovered', got %q", transition.Trigger)
	}
}

func TestMount_UpdateState_NoTransitionOnSameState(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	debounceThreshold := 3

	// First successful check - transition to Healthy
	result1 := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	transition1 := mount.UpdateState(result1, debounceThreshold)
	if transition1 == nil {
		t.Error("expected transition on first check")
	}

	// Second successful check - no transition
	result2 := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	transition2 := mount.UpdateState(result2, debounceThreshold)
	if transition2 != nil {
		t.Error("expected no transition when state unchanged")
	}
}

func TestMount_Snapshot(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	debounceThreshold := 3

	// Set some state
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   false,
		Duration:  100 * time.Millisecond,
		Error:     errors.New("connection refused"),
	}
	mount.UpdateState(result, debounceThreshold)

	snapshot := mount.Snapshot()

	if snapshot.Path != "/mnt/test" {
		t.Errorf("expected path '/mnt/test', got %q", snapshot.Path)
	}
	if snapshot.Status != health.StatusDegraded {
		t.Errorf("expected status Degraded, got %v", snapshot.Status)
	}
	if snapshot.FailureCount != 1 {
		t.Errorf("expected failure count 1, got %d", snapshot.FailureCount)
	}
	if snapshot.LastError != "connection refused" {
		t.Errorf("expected error 'connection refused', got %q", snapshot.LastError)
	}
}

func TestMount_TransientFailure_NoRestart(t *testing.T) {
	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	debounceThreshold := 3

	// Start healthy
	successResult := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	mount.UpdateState(successResult, debounceThreshold)

	// 2 failures (below threshold)
	for i := 0; i < 2; i++ {
		failResult := &health.CheckResult{
			Mount:     mount,
			Timestamp: time.Now(),
			Success:   false,
			Duration:  100 * time.Millisecond,
			Error:     errors.New("timeout"),
		}
		mount.UpdateState(failResult, debounceThreshold)
	}

	// Should be Degraded, not Unhealthy
	if mount.GetStatus() != health.StatusDegraded {
		t.Errorf("expected status Degraded (not Unhealthy) after 2 failures, got %v", mount.GetStatus())
	}

	// Recovery
	mount.UpdateState(successResult, debounceThreshold)

	if mount.GetStatus() != health.StatusHealthy {
		t.Errorf("expected status Healthy after recovery, got %v", mount.GetStatus())
	}
	if mount.GetFailureCount() != 0 {
		t.Errorf("expected failure count reset to 0, got %d", mount.GetFailureCount())
	}
}
