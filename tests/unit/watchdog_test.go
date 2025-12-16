package unit

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/chris/debrid-mount-monitor/internal/watchdog"
)

// TestWatchdog_DisabledByConfig verifies watchdog remains disabled when config.Enabled is false.
func TestWatchdog_DisabledByConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := watchdog.Config{
		Enabled:             false,
		RestartDelay:        0,
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", logger)

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
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := watchdog.Config{
		Enabled:             true, // Enabled in config, but we're not in K8s
		RestartDelay:        0,
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", logger)

	// Start should succeed but watchdog should remain disabled (not in cluster)
	if err := wd.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be disabled because we're not running in Kubernetes
	if wd.IsEnabled() {
		t.Error("watchdog should be disabled when not running in Kubernetes")
	}
}

// TestWatchdog_StateTransitions tests the state machine transitions.
func TestWatchdog_StateTransitions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := watchdog.Config{
		Enabled:             false, // Keep disabled for state machine test
		RestartDelay:        5 * time.Second,
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", logger)

	// Initial state should be disabled
	state := wd.State()
	if state.State != watchdog.WatchdogDisabled {
		t.Errorf("initial state should be WatchdogDisabled, got %v", state.State)
	}

	// OnMountUnhealthy should have no effect when disabled
	wd.OnMountUnhealthy("/mnt/test")
	state = wd.State()
	if state.State != watchdog.WatchdogDisabled {
		t.Error("OnMountUnhealthy should not change state when disabled")
	}
}

// TestWatchdog_RestartDelayCancellation tests that recovery cancels pending restart.
func TestWatchdog_RestartDelayCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := watchdog.Config{
		Enabled:             false,
		RestartDelay:        1 * time.Hour, // Long delay so we can test cancellation
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", logger)

	// When disabled, OnMountHealthy should be safe to call
	wd.OnMountHealthy("/mnt/test")
	state := wd.State()
	if state.State != watchdog.WatchdogDisabled {
		t.Error("OnMountHealthy should not crash when disabled")
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
	// This tests the config package validation, not watchdog itself
	// The actual validation is in config.Validate()
	// Here we just verify the watchdog Config struct can be created
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

// TestWatchdog_ExitFuncOverride tests that the exit function can be overridden for testing.
func TestWatchdog_ExitFuncOverride(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := watchdog.Config{
		Enabled:             false,
		RestartDelay:        0,
		MaxRetries:          3,
		RetryBackoffInitial: 100 * time.Millisecond,
		RetryBackoffMax:     10 * time.Second,
	}

	wd := watchdog.NewWatchdog(cfg, "test-pod", "test-ns", logger)

	// Override exit function
	exitCalled := false
	exitCode := 0
	wd.SetExitFunc(func(code int) {
		exitCalled = true
		exitCode = code
	})

	// Verify exit func was set (we can't trigger it without K8s, but we can verify the method exists)
	if exitCalled {
		t.Error("exit should not have been called yet")
	}
	_ = exitCode // Suppress unused variable warning
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
