package monitor_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cscheib/debrid-mount-monitor/internal/health"
	"github.com/cscheib/debrid-mount-monitor/internal/monitor"
	"github.com/matryer/is"
)

func TestMonitor_StartsAndStops(t *testing.T) {
	is := is.New(t)

	tmpDir := t.TempDir()
	canaryPath := filepath.Join(tmpDir, ".health-check")
	if err := os.WriteFile(canaryPath, []byte("ok"), 0644); err != nil {
		t.Fatalf("failed to create canary file: %v", err)
	}

	mount := health.NewMount("", tmpDir, ".health-check", 3)
	checker := health.NewChecker(5 * time.Second)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	checkInterval := 100 * time.Millisecond
	mon := monitor.New([]*health.Mount{mount}, checker, checkInterval, 3, logger)

	ctx, cancel := context.WithCancel(context.Background())
	mon.Start(ctx)

	// Poll until mount becomes healthy (more reliable than fixed sleep)
	is.True(pollForStatus(t, mount, health.StatusHealthy, 10*time.Second, checkInterval)) // mount should become healthy

	// Stop the monitor
	cancel()
	mon.Wait()
}

func TestMonitor_DetectsFailure(t *testing.T) {
	is := is.New(t)

	tmpDir := t.TempDir()
	// Don't create canary file - mount should be unhealthy

	mount := health.NewMount("", tmpDir, ".health-check", 3)
	checker := health.NewChecker(100 * time.Millisecond)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	failureThreshold := 3
	checkInterval := 50 * time.Millisecond
	mon := monitor.New([]*health.Mount{mount}, checker, checkInterval, failureThreshold, logger)

	ctx, cancel := context.WithCancel(context.Background())
	mon.Start(ctx)

	// Poll until mount becomes unhealthy (more reliable than fixed sleep)
	is.True(pollForStatus(t, mount, health.StatusUnhealthy, 10*time.Second, checkInterval)) // mount should become unhealthy

	cancel()
	mon.Wait()
}

func TestMonitor_DetectsRecovery(t *testing.T) {
	is := is.New(t)

	tmpDir := t.TempDir()
	canaryPath := filepath.Join(tmpDir, ".health-check")

	// Don't create canary initially
	mount := health.NewMount("", tmpDir, ".health-check", 3)
	checker := health.NewChecker(100 * time.Millisecond)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	checkInterval := 100 * time.Millisecond
	failureThreshold := 2
	mon := monitor.New([]*health.Mount{mount}, checker, checkInterval, failureThreshold, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		mon.Wait()
	}()
	mon.Start(ctx)

	// Poll until mount becomes unhealthy (failure threshold exceeded)
	// Use long timeout for CI with race detector
	is.True(pollForStatus(t, mount, health.StatusUnhealthy, 10*time.Second, checkInterval)) // mount should become unhealthy

	// Now create the canary file to simulate recovery
	if err := os.WriteFile(canaryPath, []byte("ok"), 0644); err != nil {
		t.Fatalf("failed to create canary file: %v", err)
	}

	// Poll for recovery with timeout - needs to be long enough for monitor to detect
	is.True(pollForStatus(t, mount, health.StatusHealthy, 10*time.Second, checkInterval)) // mount should recover
}

// pollForStatus polls for a specific mount status with timeout.
// Returns true if the status was reached, false on timeout.
func pollForStatus(t *testing.T, mount *health.Mount, expected health.HealthStatus, timeout, interval time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if mount.GetStatus() == expected {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

func TestMonitor_MultipleMount(t *testing.T) {
	is := is.New(t)

	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Mount1: healthy
	canary1 := filepath.Join(tmpDir1, ".health-check")
	if err := os.WriteFile(canary1, []byte("ok"), 0644); err != nil {
		t.Fatalf("failed to create canary file: %v", err)
	}

	// Mount2: no canary file (unhealthy)
	mount1 := health.NewMount("", tmpDir1, ".health-check", 3)
	mount2 := health.NewMount("", tmpDir2, ".health-check", 3)

	checker := health.NewChecker(100 * time.Millisecond)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	failureThreshold := 2
	checkInterval := 30 * time.Millisecond
	mounts := []*health.Mount{mount1, mount2}
	mon := monitor.New(mounts, checker, checkInterval, failureThreshold, logger)

	ctx, cancel := context.WithCancel(context.Background())
	mon.Start(ctx)

	// Poll until both mounts reach expected states (more reliable than fixed sleep)
	is.True(pollForStatus(t, mount1, health.StatusHealthy, 10*time.Second, checkInterval))   // mount1 should be healthy
	is.True(pollForStatus(t, mount2, health.StatusUnhealthy, 10*time.Second, checkInterval)) // mount2 should be unhealthy

	cancel()
	mon.Wait()
}
