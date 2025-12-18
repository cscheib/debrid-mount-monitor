package health_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/cscheib/debrid-mount-monitor/internal/health"
	"github.com/matryer/is"
)

func TestNewMount(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("test-mount", "/mnt/test", ".health-check", 3)

	is.Equal(mount.Name, "test-mount")                    // name
	is.Equal(mount.Path, "/mnt/test")                     // path
	is.Equal(mount.CanaryPath, "/mnt/test/.health-check") // canary path
	is.Equal(mount.FailureThreshold, 3)                   // failure threshold
	is.Equal(mount.GetStatus(), health.StatusUnknown)     // initial status
	is.Equal(mount.GetFailureCount(), 0)                  // initial failure count
}

func TestNewMount_TrailingSlash(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test/", ".health-check", 3)

	is.Equal(mount.CanaryPath, "/mnt/test/.health-check") // trailing slash should be handled
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
		{health.HealthStatus(99), "unknown"}, // invalid value defaults to unknown
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			is := is.New(t)
			is.Equal(tt.status.String(), tt.expected) // status string
		})
	}
}

func TestMount_UpdateState_SuccessfulCheck(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	failureThreshold := 3

	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
		Error:     nil,
	}

	transition := mount.UpdateState(result, failureThreshold)

	is.Equal(mount.GetStatus(), health.StatusHealthy)   // status should be healthy
	is.Equal(mount.GetFailureCount(), 0)                // failure count should be 0
	is.True(transition != nil)                          // should have transition
	is.Equal(transition.NewState, health.StatusHealthy) // transition to healthy
}

func TestMount_UpdateState_FailedCheck_Degraded(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	failureThreshold := 3

	// First failure - should go to Degraded
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   false,
		Duration:  100 * time.Millisecond,
		Error:     errors.New("read timeout"),
	}

	transition := mount.UpdateState(result, failureThreshold)

	is.Equal(mount.GetStatus(), health.StatusDegraded) // status should be degraded
	is.Equal(mount.GetFailureCount(), 1)               // failure count should be 1
	is.True(transition != nil)                         // should have transition
}

func TestMount_UpdateState_FailedCheck_Unhealthy(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	failureThreshold := 3

	// Simulate 3 consecutive failures
	for i := 0; i < failureThreshold; i++ {
		result := &health.CheckResult{
			Mount:     mount,
			Timestamp: time.Now(),
			Success:   false,
			Duration:  100 * time.Millisecond,
			Error:     errors.New("read timeout"),
		}
		mount.UpdateState(result, failureThreshold)
	}

	is.Equal(mount.GetStatus(), health.StatusUnhealthy) // status should be unhealthy
	is.Equal(mount.GetFailureCount(), failureThreshold) // failure count should equal threshold
}

func TestMount_UpdateState_RecoveryFromUnhealthy(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	failureThreshold := 3

	// Put mount into unhealthy state
	for i := 0; i < failureThreshold; i++ {
		result := &health.CheckResult{
			Mount:     mount,
			Timestamp: time.Now(),
			Success:   false,
			Duration:  100 * time.Millisecond,
			Error:     errors.New("read timeout"),
		}
		mount.UpdateState(result, failureThreshold)
	}

	is.Equal(mount.GetStatus(), health.StatusUnhealthy) // mount should be unhealthy

	// Recovery with successful check
	successResult := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
		Error:     nil,
	}

	transition := mount.UpdateState(successResult, failureThreshold)

	is.Equal(mount.GetStatus(), health.StatusHealthy) // status should be healthy after recovery
	is.Equal(mount.GetFailureCount(), 0)              // failure count should be 0
	is.True(transition != nil)                        // should have transition
	is.Equal(transition.Trigger, "recovered")         // trigger should be recovered
}

func TestMount_UpdateState_NoTransitionOnSameState(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	failureThreshold := 3

	// First successful check - transition to Healthy
	result1 := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	transition1 := mount.UpdateState(result1, failureThreshold)
	is.True(transition1 != nil) // should have transition on first check

	// Second successful check - no transition
	result2 := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	transition2 := mount.UpdateState(result2, failureThreshold)
	is.True(transition2 == nil) // no transition when state unchanged
}

func TestMount_Snapshot(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	failureThreshold := 3

	// Set some state
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   false,
		Duration:  100 * time.Millisecond,
		Error:     errors.New("connection refused"),
	}
	mount.UpdateState(result, failureThreshold)

	snapshot := mount.Snapshot()

	is.Equal(snapshot.Path, "/mnt/test")               // path
	is.Equal(snapshot.Status, health.StatusDegraded)   // status
	is.Equal(snapshot.FailureCount, 1)                 // failure count
	is.Equal(snapshot.LastError, "connection refused") // last error
}

func TestMount_TransientFailure_NoRestart(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)
	failureThreshold := 3

	// Start healthy
	successResult := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	mount.UpdateState(successResult, failureThreshold)

	// 2 failures (below threshold)
	for i := 0; i < 2; i++ {
		failResult := &health.CheckResult{
			Mount:     mount,
			Timestamp: time.Now(),
			Success:   false,
			Duration:  100 * time.Millisecond,
			Error:     errors.New("timeout"),
		}
		mount.UpdateState(failResult, failureThreshold)
	}

	// Should be Degraded, not Unhealthy
	is.Equal(mount.GetStatus(), health.StatusDegraded) // should be degraded after 2 failures

	// Recovery
	mount.UpdateState(successResult, failureThreshold)

	is.Equal(mount.GetStatus(), health.StatusHealthy) // should be healthy after recovery
	is.Equal(mount.GetFailureCount(), 0)              // failure count should reset
}

// TestMount_ConcurrentAccess tests that Mount is safe for concurrent reads and writes.
// This test uses the race detector (-race) to verify correctness.
func TestMount_ConcurrentAccess(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("concurrent-test", "/mnt/test", ".health-check", 3)
	failureThreshold := 3

	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 100

	// Writer goroutines - simulate monitor updating state
	for i := 0; i < goroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				result := &health.CheckResult{
					Mount:     mount,
					Timestamp: time.Now(),
					Success:   j%2 == 0, // Alternate success/failure
					Duration:  time.Duration(j) * time.Millisecond,
					Error:     nil,
				}
				if !result.Success {
					result.Error = errors.New("test error")
				}
				mount.UpdateState(result, failureThreshold)
			}
		}(i)
	}

	// Reader goroutines - simulate server reading state for health endpoints
	for i := 0; i < goroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Read operations that must be thread-safe
				_ = mount.GetStatus()
				_ = mount.GetFailureCount()
				_ = mount.Snapshot()
			}
		}(i)
	}

	wg.Wait()

	// Verify mount is in a valid state after concurrent access
	status := mount.GetStatus()
	is.True(status == health.StatusUnknown ||
		status == health.StatusHealthy ||
		status == health.StatusDegraded ||
		status == health.StatusUnhealthy) // status should be valid

	failureCount := mount.GetFailureCount()
	is.True(failureCount >= 0 && failureCount <= failureThreshold) // failure count should be valid
}

// TestMount_GetName tests the GetName method returns the mount name.
func TestMount_GetName(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("my-mount", "/mnt/test", ".health-check", 3)

	is.Equal(mount.GetName(), "my-mount") // should return mount name
}

// TestMount_GetLastCheck tests the GetLastCheck method returns the last check timestamp.
func TestMount_GetLastCheck(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)

	// Before any check, last check should be zero time
	is.True(mount.GetLastCheck().IsZero()) // no check performed yet

	// Perform a check
	checkTime := time.Now()
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: checkTime,
		Success:   true,
		Duration:  100 * time.Millisecond,
	}
	mount.UpdateState(result, 3)

	// Last check should now be set
	lastCheck := mount.GetLastCheck()
	is.True(!lastCheck.IsZero())                        // last check should be set
	is.True(lastCheck.Sub(checkTime) < time.Second)     // should be close to check time
}

// TestMount_GetLastError tests the GetLastError method returns the last error.
func TestMount_GetLastError(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/mnt/test", ".health-check", 3)

	// Before any failure, last error should be nil
	is.True(mount.GetLastError() == nil) // no error yet

	// Simulate a failure
	result := &health.CheckResult{
		Mount:     mount,
		Timestamp: time.Now(),
		Success:   false,
		Duration:  100 * time.Millisecond,
		Error:     errors.New("connection timeout"),
	}
	mount.UpdateState(result, 3)

	// Last error should now be set
	lastErr := mount.GetLastError()
	is.True(lastErr != nil)                             // error should be set
	is.Equal(lastErr.Error(), "connection timeout")     // error message should match
}
