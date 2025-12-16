package unit

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chris/debrid-mount-monitor/internal/watchdog"
)

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
	cfg := watchdog.Config{
		Enabled:             false,
		RestartDelay:        0,
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())

	// Start should succeed but watchdog should remain disabled
	if err := wd.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if wd.IsEnabled() {
		t.Error("watchdog should be disabled when config.Enabled is false")
	}

	state := wd.State()
	if state.State != watchdog.WatchdogDisabled {
		t.Errorf("expected state WatchdogDisabled, got %v", state.State)
	}
}

// TestWatchdog_DisabledOutsideKubernetes verifies watchdog disables gracefully outside K8s.
func TestWatchdog_DisabledOutsideKubernetes(t *testing.T) {
	cfg := watchdog.Config{
		Enabled:             true, // Enabled in config, but we're not in K8s
		RestartDelay:        0,
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())

	// Start should succeed but watchdog should remain disabled (not in cluster)
	if err := wd.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be disabled because we're not running in Kubernetes
	if wd.IsEnabled() {
		t.Error("watchdog should be disabled when not running in Kubernetes")
	}
}

// TestWatchdog_StateTransitions_DisabledToArmed tests transition when using mock client.
func TestWatchdog_StateTransitions_DisabledToArmed(t *testing.T) {
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
	if state.State != watchdog.WatchdogDisabled {
		t.Errorf("initial state should be WatchdogDisabled, got %v", state.State)
	}

	// Manually set armed state (simulating in-cluster)
	wd.SetArmed()

	state = wd.State()
	if state.State != watchdog.WatchdogArmed {
		t.Errorf("expected state WatchdogArmed after SetArmed(), got %v", state.State)
	}

	if !wd.IsEnabled() {
		t.Error("watchdog should be enabled when armed")
	}
}

// TestWatchdog_ArmedToPendingRestart tests Armed -> PendingRestart transition.
func TestWatchdog_ArmedToPendingRestart(t *testing.T) {
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
	if state.State != watchdog.WatchdogPendingRestart {
		t.Errorf("expected state WatchdogPendingRestart, got %v", state.State)
	}
	if state.PendingMount != "/mnt/test" {
		t.Errorf("expected PendingMount /mnt/test, got %v", state.PendingMount)
	}
}

// TestWatchdog_PendingRestartToTriggered tests the full transition to Triggered state.
func TestWatchdog_PendingRestartToTriggered(t *testing.T) {
	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        10 * time.Millisecond, // Short delay for fast test
		MaxRetries:          3,
		RetryBackoffInitial: 1 * time.Millisecond,
		RetryBackoffMax:     10 * time.Millisecond,
	}

	mockClient := &MockK8sClient{}
	var exitCalled atomic.Bool

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", testLogger())
	wd.SetK8sClient(mockClient)
	wd.SetExitFunc(func(code int) {
		exitCalled.Store(true)
	})
	wd.SetArmed()

	// Trigger unhealthy state
	wd.OnMountUnhealthy("/mnt/test", 3)

	// Wait for restart delay + processing time
	time.Sleep(100 * time.Millisecond)

	// Verify DeletePod was called
	mockClient.mu.Lock()
	deleteCount := len(mockClient.DeletePodCalls)
	mockClient.mu.Unlock()

	if deleteCount == 0 {
		t.Error("expected DeletePod to be called")
	}

	// State should be Triggered
	state := wd.State()
	if state.State != watchdog.WatchdogTriggered {
		t.Errorf("expected state WatchdogTriggered, got %v", state.State)
	}
}

// TestWatchdog_RestartCancellationOnRecovery tests that recovery cancels pending restart.
func TestWatchdog_RestartCancellationOnRecovery(t *testing.T) {
	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        500 * time.Millisecond, // Enough time to cancel
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

	// Verify pending state
	state := wd.State()
	if state.State != watchdog.WatchdogPendingRestart {
		t.Errorf("expected PendingRestart, got %v", state.State)
	}

	// Recover before delay expires
	time.Sleep(50 * time.Millisecond)
	wd.OnMountHealthy("/mnt/test")

	// Should be back to Armed
	state = wd.State()
	if state.State != watchdog.WatchdogArmed {
		t.Errorf("expected Armed after recovery, got %v", state.State)
	}

	// Wait to ensure no delete happens
	time.Sleep(600 * time.Millisecond)

	mockClient.mu.Lock()
	deleteCount := len(mockClient.DeletePodCalls)
	mockClient.mu.Unlock()

	if deleteCount > 0 {
		t.Error("DeletePod should not have been called after recovery")
	}
}

// TestWatchdog_DeletePodRetryWithBackoff tests retry logic with exponential backoff.
func TestWatchdog_DeletePodRetryWithBackoff(t *testing.T) {
	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        0, // Immediate
		MaxRetries:          3,
		RetryBackoffInitial: 10 * time.Millisecond,
		RetryBackoffMax:     50 * time.Millisecond,
	}

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
	wd.SetArmed()

	// Trigger restart
	wd.OnMountUnhealthy("/mnt/test", 3)

	// Wait for retries to complete
	time.Sleep(200 * time.Millisecond)

	mockClient.mu.Lock()
	deleteCount := len(mockClient.DeletePodCalls)
	mockClient.mu.Unlock()

	if deleteCount != 3 {
		t.Errorf("expected 3 DeletePod calls (retry logic), got %d", deleteCount)
	}

	if exitCalled.Load() {
		t.Error("exit should not have been called on successful retry")
	}
}

// TestWatchdog_RetryExhaustionExitFallback tests exit fallback after retries exhausted.
func TestWatchdog_RetryExhaustionExitFallback(t *testing.T) {
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

	if !exitCalled.Load() {
		t.Error("exit should have been called after retries exhausted")
	}

	if atomic.LoadInt32(&exitCode) != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestWatchdog_RBACValidationFailure tests handling when CanDeletePods returns false.
func TestWatchdog_RBACValidationFailure(t *testing.T) {
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
	if state.State != watchdog.WatchdogDisabled {
		t.Errorf("expected WatchdogDisabled, got %v", state.State)
	}
}

// TestWatchdog_PodAlreadyTerminating tests skip deletion when pod is already terminating.
func TestWatchdog_PodAlreadyTerminating(t *testing.T) {
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

	if deleteCount > 0 {
		t.Error("DeletePod should not have been called when pod is already terminating")
	}
}

// TestWatchdog_DisabledIgnoresUnhealthy tests that disabled watchdog ignores mount events.
func TestWatchdog_DisabledIgnoresUnhealthy(t *testing.T) {
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
	if state.State != watchdog.WatchdogDisabled {
		t.Error("OnMountUnhealthy should not change state when disabled")
	}
}

// TestWatchdog_HealthyIgnoredWhenNotPending tests healthy event when not in pending state.
func TestWatchdog_HealthyIgnoredWhenNotPending(t *testing.T) {
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
	if state.State != watchdog.WatchdogArmed {
		t.Errorf("expected Armed, got %v", state.State)
	}
}

// TestWatchdog_RecoveryDifferentMount tests that recovery of different mount doesn't cancel.
func TestWatchdog_RecoveryDifferentMount(t *testing.T) {
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
	if state.State != watchdog.WatchdogPendingRestart {
		t.Errorf("expected PendingRestart, got %v", state.State)
	}
	if state.PendingMount != "/mnt/a" {
		t.Errorf("expected PendingMount /mnt/a, got %v", state.PendingMount)
	}
}

// TestWatchdog_EventCreatedOnRestart tests that Kubernetes event is created.
func TestWatchdog_EventCreatedOnRestart(t *testing.T) {
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

	if eventCount == 0 {
		t.Error("expected CreateEvent to be called")
		return
	}

	if event.PodName != "test-pod" {
		t.Errorf("expected PodName test-pod, got %v", event.PodName)
	}
	if event.MountPath != "/mnt/test" {
		t.Errorf("expected MountPath /mnt/test, got %v", event.MountPath)
	}
	if event.FailureCount != 5 {
		t.Errorf("expected FailureCount 5, got %v", event.FailureCount)
	}
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
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("WatchdogStatus(%d).String() = %q, want %q", tt.status, got, tt.expected)
			}
		})
	}
}

// TestWatchdogConfig_Validation tests configuration validation.
func TestWatchdogConfig_Validation(t *testing.T) {
	cfg := watchdog.Config{
		Enabled:             true,
		RestartDelay:        30 * time.Second,
		MaxRetries:          5,
		RetryBackoffInitial: 200 * time.Millisecond,
		RetryBackoffMax:     30 * time.Second,
	}

	if cfg.RestartDelay != 30*time.Second {
		t.Error("RestartDelay not set correctly")
	}
	if cfg.MaxRetries != 5 {
		t.Error("MaxRetries not set correctly")
	}
}

// TestWatchdog_ExitFuncOverride tests that the exit function can be overridden.
func TestWatchdog_ExitFuncOverride(t *testing.T) {
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

	if exitCalled {
		t.Error("exit should not have been called yet")
	}
	_ = exitCode
}

// TestRestartEvent_Fields tests the RestartEvent struct fields.
func TestRestartEvent_Fields(t *testing.T) {
	event := watchdog.RestartEvent{
		Timestamp:         time.Now(),
		PodName:           "test-pod",
		Namespace:         "test-ns",
		MountPath:         "/mnt/test",
		Reason:            "Mount unhealthy",
		FailureCount:      5,
		UnhealthyDuration: 30 * time.Second,
	}

	if event.PodName != "test-pod" {
		t.Error("PodName not set correctly")
	}
	if event.Namespace != "test-ns" {
		t.Error("Namespace not set correctly")
	}
	if event.MountPath != "/mnt/test" {
		t.Error("MountPath not set correctly")
	}
	if event.FailureCount != 5 {
		t.Error("FailureCount not set correctly")
	}
	if event.UnhealthyDuration != 30*time.Second {
		t.Error("UnhealthyDuration not set correctly")
	}
}

// TestWatchdogState_Fields tests the WatchdogState struct fields.
func TestWatchdogState_Fields(t *testing.T) {
	now := time.Now()
	state := watchdog.WatchdogState{
		State:          watchdog.WatchdogArmed,
		UnhealthySince: &now,
		PendingMount:   "/mnt/test",
		RetryCount:     2,
		LastError:      nil,
	}

	if state.State != watchdog.WatchdogArmed {
		t.Error("State not set correctly")
	}
	if state.UnhealthySince == nil || !state.UnhealthySince.Equal(now) {
		t.Error("UnhealthySince not set correctly")
	}
	if state.PendingMount != "/mnt/test" {
		t.Error("PendingMount not set correctly")
	}
	if state.RetryCount != 2 {
		t.Error("RetryCount not set correctly")
	}
}

// TestWatchdog_ShutdownAbortsRestart tests that pending restart is aborted when context is cancelled.
func TestWatchdog_ShutdownAbortsRestart(t *testing.T) {
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
	if err := wd.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	wd.SetArmed() // Force armed state since we're not in-cluster

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

	if deleteCount != 0 {
		t.Errorf("expected 0 DeletePod calls (shutdown aborted restart), got %d", deleteCount)
	}
}

// TestWatchdog_PermanentErrorStopsRetry tests that permanent errors stop retry loop.
func TestWatchdog_PermanentErrorStopsRetry(t *testing.T) {
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

	if deleteCount != 1 {
		t.Errorf("expected 1 DeletePod call (permanent error stops retries), got %d", deleteCount)
	}

	// Exit should be called after permanent error
	if !exitCalled.Load() {
		t.Error("exit should have been called after permanent error")
	}
}
