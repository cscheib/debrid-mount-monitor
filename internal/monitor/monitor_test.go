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

	mon := monitor.New([]*health.Mount{mount}, checker, 100*time.Millisecond, 3, logger)

	ctx, cancel := context.WithCancel(context.Background())
	mon.Start(ctx)

	// Give it time to run at least one check
	time.Sleep(150 * time.Millisecond)

	// Stop the monitor
	cancel()
	mon.Wait()

	// Verify the mount was checked and is healthy
	is.Equal(mount.GetStatus(), health.StatusHealthy) // mount should be healthy after check
}

func TestMonitor_DetectsFailure(t *testing.T) {
	is := is.New(t)

	tmpDir := t.TempDir()
	// Don't create canary file - mount should be unhealthy

	mount := health.NewMount("", tmpDir, ".health-check", 3)
	checker := health.NewChecker(100 * time.Millisecond)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	failureThreshold := 3
	mon := monitor.New([]*health.Mount{mount}, checker, 50*time.Millisecond, failureThreshold, logger)

	ctx, cancel := context.WithCancel(context.Background())
	mon.Start(ctx)

	// Wait for enough checks to exceed failure threshold
	time.Sleep(250 * time.Millisecond)

	cancel()
	mon.Wait()

	// Verify the mount transitioned to unhealthy
	is.Equal(mount.GetStatus(), health.StatusUnhealthy) // mount should be unhealthy after failures
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
	mounts := []*health.Mount{mount1, mount2}
	mon := monitor.New(mounts, checker, 30*time.Millisecond, failureThreshold, logger)

	ctx, cancel := context.WithCancel(context.Background())
	mon.Start(ctx)

	// Wait for enough checks to exceed failure threshold for mount2
	// Need at least failureThreshold+1 intervals (initial check + threshold failures)
	time.Sleep(time.Duration(failureThreshold+2) * 50 * time.Millisecond)

	cancel()
	mon.Wait()

	// Mount1 should be healthy
	is.Equal(mount1.GetStatus(), health.StatusHealthy) // mount1 should be healthy

	// Mount2 should be unhealthy (at least degraded due to no canary)
	status2 := mount2.GetStatus()
	is.True(status2 == health.StatusUnhealthy || status2 == health.StatusDegraded) // mount2 should be unhealthy or degraded
}
