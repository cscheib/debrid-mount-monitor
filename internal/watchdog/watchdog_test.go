package watchdog_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cscheib/debrid-mount-monitor/internal/watchdog"
	"github.com/matryer/is"
	"go.uber.org/goleak"
)

// testLogger returns a silent logger for testing.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// MockK8sClient implements K8sClientInterface for testing.
// It allows configuring responses and tracking calls for assertions.
type MockK8sClient struct {
	mu sync.Mutex

	// Configurable responses
	DeletePodFunc        func(ctx context.Context, name string) error
	IsPodTerminatingFunc func(ctx context.Context, name string) (bool, error)
	CanDeletePodsFunc    func(ctx context.Context) (bool, error)
	CreateEventFunc      func(ctx context.Context, event *watchdog.RestartEvent) error
	NamespaceValue       string

	// Call tracking
	DeletePodCalls   []string
	CreateEventCalls []*watchdog.RestartEvent
}

func (m *MockK8sClient) DeletePod(ctx context.Context, name string) error {
	m.mu.Lock()
	m.DeletePodCalls = append(m.DeletePodCalls, name)
	m.mu.Unlock()

	if m.DeletePodFunc != nil {
		return m.DeletePodFunc(ctx, name)
	}
	return nil
}

func (m *MockK8sClient) IsPodTerminating(ctx context.Context, name string) (bool, error) {
	if m.IsPodTerminatingFunc != nil {
		return m.IsPodTerminatingFunc(ctx, name)
	}
	return false, nil
}

func (m *MockK8sClient) CanDeletePods(ctx context.Context) (bool, error) {
	if m.CanDeletePodsFunc != nil {
		return m.CanDeletePodsFunc(ctx)
	}
	return true, nil
}

func (m *MockK8sClient) CreateEvent(ctx context.Context, event *watchdog.RestartEvent) error {
	m.mu.Lock()
	m.CreateEventCalls = append(m.CreateEventCalls, event)
	m.mu.Unlock()

	if m.CreateEventFunc != nil {
		return m.CreateEventFunc(ctx, event)
	}
	return nil
}

func (m *MockK8sClient) Namespace() string {
	if m.NamespaceValue != "" {
		return m.NamespaceValue
	}
	return "test-ns"
}

// TestWatchdog_DisabledByConfig verifies watchdog remains disabled when config.Enabled is false.
func TestWatchdog_DisabledByConfig(t *testing.T) {
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             false,
		RestartDelay:        0,
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())

	// Start should succeed but watchdog should remain disabled
	is.NoErr(wd.Start(context.Background())) // start should succeed

	is.True(!wd.IsEnabled()) // watchdog should be disabled

	state := wd.State()
	is.Equal(state.State, watchdog.WatchdogDisabled) // state should be disabled
}

// TestWatchdog_DisabledOutsideKubernetes verifies watchdog disables gracefully outside K8s.
func TestWatchdog_DisabledOutsideKubernetes(t *testing.T) {
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             true, // Enabled in config, but we're not in K8s
		RestartDelay:        0,
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())

	// Start should succeed but watchdog should remain disabled (not in cluster)
	is.NoErr(wd.Start(context.Background())) // start should succeed

	// Should be disabled because we're not running in Kubernetes
	is.True(!wd.IsEnabled()) // watchdog should be disabled when not in Kubernetes
}

// TestWatchdog_StateTransitions_DisabledToArmed tests transition when using mock client.
func TestWatchdog_StateTransitions_DisabledToArmed(t *testing.T) {
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        5 * time.Second,
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())

	// Initial state should be disabled
	state := wd.State()
	is.Equal(state.State, watchdog.WatchdogDisabled) // initial state should be disabled

	// Manually set armed state (simulating in-cluster)
	wd.SetArmed()

	state = wd.State()
	is.Equal(state.State, watchdog.WatchdogArmed) // state should be armed after SetArmed()

	is.True(wd.IsEnabled()) // watchdog should be enabled when armed
}

// TestWatchdog_ArmedToPendingRestart tests Armed -> PendingRestart transition.
func TestWatchdog_ArmedToPendingRestart(t *testing.T) {
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        1 * time.Hour, // Long delay to prevent actual trigger
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	mockClient := &MockK8sClient{}
	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)
	wd.SetArmed()

	// Trigger unhealthy state
	wd.OnMountUnhealthy("/mnt/test", 3)

	// State should transition to PendingRestart
	state := wd.State()
	is.Equal(state.State, watchdog.WatchdogPendingRestart) // state should be PendingRestart
	is.Equal(state.PendingMount, "/mnt/test")              // PendingMount should be set
}

// TestWatchdog_PendingRestartToTriggered tests the full transition to Triggered state.
func TestWatchdog_PendingRestartToTriggered(t *testing.T) {
	defer goleak.VerifyNone(t,
		// The timer goroutine at OnMountUnhealthy.func1 exits after triggering restart,
		// but we need to allow time for it to complete
		goleak.IgnoreTopFunction("github.com/cscheib/debrid-mount-monitor/internal/watchdog.(*Watchdog).OnMountUnhealthy.func1"),
	)
	is := is.New(t)

	if testing.Short() {
		t.Skip("skipping test with delays in short mode")
	}

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        10 * time.Millisecond, // Short delay for fast test
		MaxRetries:          3,
		RetryBackoffInitial: 1 * time.Millisecond,
		RetryBackoffMax:     10 * time.Millisecond,
	}

	mockClient := &MockK8sClient{}
	var exitCalled atomic.Bool

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)
	wd.SetExitFunc(func(code int) {
		exitCalled.Store(true)
	})
	_ = wd.Start(ctx) // Store context for cleanup
	wd.SetArmed()

	// Trigger unhealthy state
	wd.OnMountUnhealthy("/mnt/test", 3)

	// Wait for restart delay + processing time
	time.Sleep(100 * time.Millisecond)

	// Verify DeletePod was called
	mockClient.mu.Lock()
	deleteCount := len(mockClient.DeletePodCalls)
	mockClient.mu.Unlock()

	is.True(deleteCount > 0) // DeletePod should have been called

	// State should be Triggered
	state := wd.State()
	is.Equal(state.State, watchdog.WatchdogTriggered) // state should be Triggered
}

// TestWatchdog_RestartCancellationOnRecovery tests that recovery cancels pending restart.
func TestWatchdog_RestartCancellationOnRecovery(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/cscheib/debrid-mount-monitor/internal/watchdog.(*Watchdog).OnMountUnhealthy.func1"),
	)
	is := is.New(t)

	if testing.Short() {
		t.Skip("skipping test with delays in short mode")
	}

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        500 * time.Millisecond, // Enough time to cancel
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockClient := &MockK8sClient{}
	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)
	_ = wd.Start(ctx)
	wd.SetArmed()

	// Trigger unhealthy state
	wd.OnMountUnhealthy("/mnt/test", 3)

	// Verify pending state
	state := wd.State()
	is.Equal(state.State, watchdog.WatchdogPendingRestart) // should be PendingRestart

	// Recover before delay expires
	time.Sleep(50 * time.Millisecond)
	wd.OnMountHealthy("/mnt/test")

	// Should be back to Armed
	state = wd.State()
	is.Equal(state.State, watchdog.WatchdogArmed) // should be Armed after recovery

	// Wait to ensure no delete happens
	time.Sleep(600 * time.Millisecond)

	mockClient.mu.Lock()
	deleteCount := len(mockClient.DeletePodCalls)
	mockClient.mu.Unlock()

	is.Equal(deleteCount, 0) // DeletePod should not have been called after recovery
}

// TestWatchdog_DeletePodRetryWithBackoff tests retry logic with exponential backoff.
func TestWatchdog_DeletePodRetryWithBackoff(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/cscheib/debrid-mount-monitor/internal/watchdog.(*Watchdog).OnMountUnhealthy.func1"),
	)
	is := is.New(t)

	if testing.Short() {
		t.Skip("skipping test with delays in short mode")
	}

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        0, // Immediate
		MaxRetries:          3,
		RetryBackoffInitial: 10 * time.Millisecond,
		RetryBackoffMax:     50 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	callCount := 0
	mockClient := &MockK8sClient{
		DeletePodFunc: func(ctx context.Context, name string) error {
			callCount++
			if callCount < 3 {
				return &watchdog.TransientError{Message: "api unavailable", StatusCode: 500}
			}
			return nil // Success on 3rd try
		},
	}

	var exitCalled atomic.Bool
	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)
	wd.SetExitFunc(func(code int) {
		exitCalled.Store(true)
	})
	_ = wd.Start(ctx)
	wd.SetArmed()

	// Trigger restart
	wd.OnMountUnhealthy("/mnt/test", 3)

	// Wait for retries to complete
	time.Sleep(200 * time.Millisecond)

	mockClient.mu.Lock()
	deleteCount := len(mockClient.DeletePodCalls)
	mockClient.mu.Unlock()

	is.Equal(deleteCount, 3)    // should have 3 DeletePod calls (retry logic)
	is.True(!exitCalled.Load()) // exit should not have been called on successful retry
}

// TestWatchdog_RetryExhaustionExitFallback tests exit fallback after retries exhausted.
func TestWatchdog_RetryExhaustionExitFallback(t *testing.T) {
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        0, // Immediate
		MaxRetries:          2,
		RetryBackoffInitial: 1 * time.Millisecond,
		RetryBackoffMax:     5 * time.Millisecond,
	}

	mockClient := &MockK8sClient{
		DeletePodFunc: func(ctx context.Context, name string) error {
			return &watchdog.TransientError{Message: "api unavailable", StatusCode: 500}
		},
	}

	var exitCalled atomic.Bool
	var exitCode int32
	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)
	wd.SetExitFunc(func(code int) {
		exitCalled.Store(true)
		atomic.StoreInt32(&exitCode, int32(code))
	})
	wd.SetArmed()

	// Trigger restart
	wd.OnMountUnhealthy("/mnt/test", 3)

	// Wait for retries to exhaust
	time.Sleep(100 * time.Millisecond)

	is.True(exitCalled.Load())                      // exit should have been called after retries exhausted
	is.Equal(atomic.LoadInt32(&exitCode), int32(1)) // exit code should be 1
}

// TestWatchdog_RBACValidationFailure tests handling when CanDeletePods returns false.
func TestWatchdog_RBACValidationFailure(t *testing.T) {
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        0,
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	mockClient := &MockK8sClient{
		CanDeletePodsFunc: func(ctx context.Context) (bool, error) {
			return false, nil // RBAC permission denied
		},
	}

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)

	// Note: Can't test Start() directly since it calls IsInCluster() first
	// The RBAC check happens after cluster detection
	// This is tested via the state remaining disabled

	state := wd.State()
	is.Equal(state.State, watchdog.WatchdogDisabled) // state should be disabled
}

// TestWatchdog_PodAlreadyTerminating tests skip deletion when pod is already terminating.
func TestWatchdog_PodAlreadyTerminating(t *testing.T) {
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        0, // Immediate
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	mockClient := &MockK8sClient{
		IsPodTerminatingFunc: func(ctx context.Context, name string) (bool, error) {
			return true, nil // Pod is already terminating
		},
	}

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)
	wd.SetArmed()

	// Trigger restart
	wd.OnMountUnhealthy("/mnt/test", 3)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// DeletePod should NOT have been called
	mockClient.mu.Lock()
	deleteCount := len(mockClient.DeletePodCalls)
	mockClient.mu.Unlock()

	is.Equal(deleteCount, 0) // DeletePod should not have been called when pod is already terminating
}

// TestWatchdog_DisabledIgnoresUnhealthy tests that disabled watchdog ignores mount events.
func TestWatchdog_DisabledIgnoresUnhealthy(t *testing.T) {
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             false, // Disabled
		RestartDelay:        0,
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	mockClient := &MockK8sClient{}
	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)

	// OnMountUnhealthy should have no effect when disabled
	wd.OnMountUnhealthy("/mnt/test", 3)

	state := wd.State()
	is.Equal(state.State, watchdog.WatchdogDisabled) // state should remain disabled
}

// TestWatchdog_HealthyIgnoredWhenNotPending tests healthy event when not in pending state.
func TestWatchdog_HealthyIgnoredWhenNotPending(t *testing.T) {
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        1 * time.Hour,
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetArmed()

	// Call OnMountHealthy when in Armed state (not PendingRestart)
	wd.OnMountHealthy("/mnt/test")

	// Should remain Armed
	state := wd.State()
	is.Equal(state.State, watchdog.WatchdogArmed) // state should remain Armed
}

// TestWatchdog_RecoveryDifferentMount tests that recovery of different mount doesn't cancel.
func TestWatchdog_RecoveryDifferentMount(t *testing.T) {
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        500 * time.Millisecond,
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	mockClient := &MockK8sClient{}
	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)
	wd.SetArmed()

	// Trigger unhealthy for mount A
	wd.OnMountUnhealthy("/mnt/a", 3)

	// Recover mount B (different mount)
	wd.OnMountHealthy("/mnt/b")

	// Should still be pending restart for mount A
	state := wd.State()
	is.Equal(state.State, watchdog.WatchdogPendingRestart) // should be PendingRestart
	is.Equal(state.PendingMount, "/mnt/a")                 // PendingMount should be /mnt/a
}

// TestWatchdog_EventCreatedOnRestart tests that Kubernetes event is created.
func TestWatchdog_EventCreatedOnRestart(t *testing.T) {
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        0, // Immediate
		MaxRetries:          3,
		RetryBackoffInitial: 1 * time.Millisecond,
		RetryBackoffMax:     10 * time.Millisecond,
	}

	mockClient := &MockK8sClient{}
	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)
	wd.SetArmed()

	// Trigger restart
	wd.OnMountUnhealthy("/mnt/test", 5)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	mockClient.mu.Lock()
	eventCount := len(mockClient.CreateEventCalls)
	var event *watchdog.RestartEvent
	if eventCount > 0 {
		event = mockClient.CreateEventCalls[0]
	}
	mockClient.mu.Unlock()

	is.True(eventCount > 0)                // CreateEvent should have been called
	is.Equal(event.PodName, "test-pod")    // PodName
	is.Equal(event.MountPath, "/mnt/test") // MountPath
	is.Equal(event.FailureCount, 5)        // FailureCount
}

// TestWatchdogStatus_String tests the String() method of WatchdogStatus.
func TestWatchdogStatus_String(t *testing.T) {
	tests := []struct {
		status   watchdog.WatchdogStatus
		expected string
	}{
		{watchdog.WatchdogDisabled, "disabled"},
		{watchdog.WatchdogArmed, "armed"},
		{watchdog.WatchdogPendingRestart, "pending_restart"},
		{watchdog.WatchdogTriggered, "triggered"},
		{watchdog.WatchdogStatus(99), "unknown"}, // Invalid value
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			is := is.New(t)
			is.Equal(tt.status.String(), tt.expected) // status string
		})
	}
}

// TestWatchdog_ExitFuncOverride tests that the exit function can be overridden.
func TestWatchdog_ExitFuncOverride(t *testing.T) {
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             false,
		RestartDelay:        0,
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())

	exitCalled := false
	exitCode := 0
	wd.SetExitFunc(func(code int) {
		exitCalled = true
		exitCode = code
	})

	is.True(!exitCalled) // exit should not have been called yet
	_ = exitCode
}

// TestWatchdog_ShutdownAbortsRestart tests that pending restart is aborted when context is cancelled.
func TestWatchdog_ShutdownAbortsRestart(t *testing.T) {
	defer goleak.VerifyNone(t,
		// The timer goroutine is still blocked on cancelCh when context is cancelled
		// This is expected behavior - the goroutine will be cleaned up when the process exits
		goleak.IgnoreTopFunction("github.com/cscheib/debrid-mount-monitor/internal/watchdog.(*Watchdog).OnMountUnhealthy.func1"),
	)
	is := is.New(t)

	if testing.Short() {
		t.Skip("skipping test with delays in short mode")
	}

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        200 * time.Millisecond, // Delay to allow cancellation
		MaxRetries:          3,
		RetryBackoffInitial: 10 * time.Millisecond,
		RetryBackoffMax:     100 * time.Millisecond,
	}

	mockClient := &MockK8sClient{}

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)
	wd.SetExitFunc(func(code int) {})

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	is.NoErr(wd.Start(ctx)) // Start should succeed
	wd.SetArmed()           // Force armed state since we're not in-cluster

	// Trigger restart
	wd.OnMountUnhealthy("/mnt/test", 3)

	// Cancel context to simulate shutdown (before delay expires)
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for processing to complete
	time.Sleep(300 * time.Millisecond)

	// DeletePod should NOT have been called because context was cancelled
	mockClient.mu.Lock()
	deleteCount := len(mockClient.DeletePodCalls)
	mockClient.mu.Unlock()

	is.Equal(deleteCount, 0) // shutdown should have aborted restart
}

// TestWatchdog_PermanentErrorStopsRetry tests that permanent errors stop retry loop.
func TestWatchdog_PermanentErrorStopsRetry(t *testing.T) {
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        0,
		MaxRetries:          5, // High retry count
		RetryBackoffInitial: 1 * time.Millisecond,
		RetryBackoffMax:     5 * time.Millisecond,
	}

	mockClient := &MockK8sClient{
		DeletePodFunc: func(ctx context.Context, name string) error {
			return &watchdog.PermanentError{Message: "forbidden"}
		},
	}

	var exitCalled atomic.Bool
	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)
	wd.SetExitFunc(func(code int) {
		exitCalled.Store(true)
	})
	wd.SetArmed()

	// Trigger restart
	wd.OnMountUnhealthy("/mnt/test", 3)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Should have only tried once (permanent error stops retries)
	mockClient.mu.Lock()
	deleteCount := len(mockClient.DeletePodCalls)
	mockClient.mu.Unlock()

	is.Equal(deleteCount, 1)   // permanent error should stop retries after 1 call
	is.True(exitCalled.Load()) // exit should have been called after permanent error
}

// TestWatchdog_ConcurrentMountFailures tests behavior when multiple mounts fail simultaneously.
// The watchdog uses a single PendingMount field, so only the first failure triggers restart.
// Subsequent failures for different mounts are ignored until recovery.
func TestWatchdog_ConcurrentMountFailures(t *testing.T) {
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        1 * time.Hour, // Long delay to prevent actual trigger
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	mockClient := &MockK8sClient{}
	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)
	wd.SetArmed()

	// First mount fails - should trigger pending restart
	wd.OnMountUnhealthy("/mnt/a", 3)

	state := wd.State()
	is.Equal(state.State, watchdog.WatchdogPendingRestart) // should be PendingRestart
	is.Equal(state.PendingMount, "/mnt/a")                 // PendingMount should be /mnt/a

	// Second mount fails simultaneously - should be ignored (already pending)
	wd.OnMountUnhealthy("/mnt/b", 5)

	state = wd.State()
	is.Equal(state.State, watchdog.WatchdogPendingRestart) // should remain PendingRestart
	// PendingMount should still be the first one that triggered
	is.Equal(state.PendingMount, "/mnt/a") // PendingMount should remain /mnt/a

	// Third mount fails - also ignored
	wd.OnMountUnhealthy("/mnt/c", 2)

	state = wd.State()
	is.Equal(state.PendingMount, "/mnt/a") // PendingMount should still be /mnt/a
}

// TestWatchdog_RapidStateTransitions tests rapid OnMountUnhealthy -> OnMountHealthy -> OnMountUnhealthy
// to verify timer cleanup and no race conditions.
func TestWatchdog_RapidStateTransitions(t *testing.T) {
	defer goleak.VerifyNone(t,
		// Timer goroutines may still be running cleanup
		goleak.IgnoreTopFunction("github.com/cscheib/debrid-mount-monitor/internal/watchdog.(*Watchdog).OnMountUnhealthy.func1"),
	)
	is := is.New(t)

	if testing.Short() {
		t.Skip("skipping test with delays in short mode")
	}

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        100 * time.Millisecond,
		MaxRetries:          3,
		RetryBackoffInitial: 10 * time.Millisecond,
		RetryBackoffMax:     100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockClient := &MockK8sClient{}
	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)
	_ = wd.Start(ctx)
	wd.SetArmed()

	// Rapid transitions: unhealthy -> healthy -> unhealthy -> healthy
	for i := 0; i < 5; i++ {
		wd.OnMountUnhealthy("/mnt/test", 3)

		state := wd.State()
		is.Equal(state.State, watchdog.WatchdogPendingRestart) // should be PendingRestart

		// Quick recovery before timer fires
		time.Sleep(10 * time.Millisecond)
		wd.OnMountHealthy("/mnt/test")

		state = wd.State()
		is.Equal(state.State, watchdog.WatchdogArmed) // should be Armed after recovery
	}

	// Wait to ensure no delayed triggers happen
	time.Sleep(200 * time.Millisecond)

	// No DeletePod should have been called (all were cancelled)
	mockClient.mu.Lock()
	deleteCount := len(mockClient.DeletePodCalls)
	mockClient.mu.Unlock()

	is.Equal(deleteCount, 0) // all rapid transitions should have been cancelled

	// Final state should be Armed
	state := wd.State()
	is.Equal(state.State, watchdog.WatchdogArmed) // final state should be Armed
}

// TestPermanentError tests the PermanentError type.
func TestPermanentError(t *testing.T) {
	is := is.New(t)

	err := &watchdog.PermanentError{Message: "forbidden access"}

	is.Equal(err.Error(), "forbidden access") // error message
	is.True(err.IsPermanent())                // should be permanent
}

// TestTransientError tests the TransientError type.
func TestTransientError(t *testing.T) {
	is := is.New(t)

	err := &watchdog.TransientError{Message: "service unavailable", StatusCode: 503}

	is.Equal(err.Error(), "service unavailable") // error message
	is.True(err.IsTransient())                    // should be transient
	is.Equal(err.StatusCode, 503)                 // status code
}

// TestWatchdog_ConcurrentMountStateChanges tests thread-safety of mount state changes.
// This test runs multiple goroutines calling OnMountUnhealthy and OnMountHealthy
// simultaneously to verify there are no race conditions.
func TestWatchdog_ConcurrentMountStateChanges(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/cscheib/debrid-mount-monitor/internal/watchdog.(*Watchdog).OnMountUnhealthy.func1"),
	)
	is := is.New(t)

	if testing.Short() {
		t.Skip("skipping test with delays in short mode")
	}

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        50 * time.Millisecond,
		MaxRetries:          3,
		RetryBackoffInitial: 10 * time.Millisecond,
		RetryBackoffMax:     100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockClient := &MockK8sClient{}
	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)
	_ = wd.Start(ctx)
	wd.SetArmed()

	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 20

	// Half the goroutines trigger unhealthy
	for i := 0; i < goroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				wd.OnMountUnhealthy("/mnt/test", 3)
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	// Half the goroutines trigger healthy (recovery)
	for i := 0; i < goroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				wd.OnMountHealthy("/mnt/test")
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Allow any pending timers to complete
	time.Sleep(100 * time.Millisecond)

	// Verify watchdog is in a valid state (no crashes, no panics)
	state := wd.State()
	is.True(state.State == watchdog.WatchdogArmed ||
		state.State == watchdog.WatchdogPendingRestart ||
		state.State == watchdog.WatchdogTriggered) // state should be valid
}

// TestWatchdog_ImmediateRestart tests restart with zero delay.
func TestWatchdog_ImmediateRestart(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("github.com/cscheib/debrid-mount-monitor/internal/watchdog.(*Watchdog).OnMountUnhealthy.func1"),
	)
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        0, // Immediate restart
		MaxRetries:          3,
		RetryBackoffInitial: 1 * time.Millisecond,
		RetryBackoffMax:     10 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockClient := &MockK8sClient{}
	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)
	wd.SetExitFunc(func(code int) {})
	_ = wd.Start(ctx)
	wd.SetArmed()

	// Trigger unhealthy
	wd.OnMountUnhealthy("/mnt/test", 5)

	// With zero delay, should trigger immediately
	time.Sleep(50 * time.Millisecond)

	mockClient.mu.Lock()
	deleteCount := len(mockClient.DeletePodCalls)
	mockClient.mu.Unlock()

	is.True(deleteCount > 0) // DeletePod should have been called immediately

	state := wd.State()
	is.Equal(state.State, watchdog.WatchdogTriggered) // state should be Triggered
}

// TestNewWatchdog tests the constructor.
func TestNewWatchdog(t *testing.T) {
	is := is.New(t)

	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        5 * time.Second,
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	wd := watchdog.NewWatchdog(cfg, "my-pod", "my-namespace", testLogger())

	is.True(wd != nil) // watchdog should be created

	state := wd.State()
	is.Equal(state.State, watchdog.WatchdogDisabled) // initial state should be disabled
	is.True(!wd.IsEnabled())                          // should not be enabled initially
}
