package unit

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chris/debrid-mount-monitor/internal/health"
	"github.com/chris/debrid-mount-monitor/internal/monitor"
)

func TestMonitor_StartsAndStops(t *testing.T) {
	tmpDir := t.TempDir()
	canaryPath := filepath.Join(tmpDir, ".health-check")
	if err := os.WriteFile(canaryPath, []byte("ok"), 0644); err != nil {
		t.Fatalf("failed to create canary file: %v", err)
	}

	mount := health.NewMount(tmpDir, ".health-check")
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
	if mount.GetStatus() != health.StatusHealthy {
		t.Errorf("expected mount status Healthy after check, got %v", mount.GetStatus())
	}
}

func TestMonitor_DetectsFailure(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create canary file - mount should be unhealthy

	mount := health.NewMount(tmpDir, ".health-check")
	checker := health.NewChecker(100 * time.Millisecond)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	debounceThreshold := 3
	mon := monitor.New([]*health.Mount{mount}, checker, 50*time.Millisecond, debounceThreshold, logger)

	ctx, cancel := context.WithCancel(context.Background())
	mon.Start(ctx)

	// Wait for enough checks to exceed debounce threshold
	time.Sleep(250 * time.Millisecond)

	cancel()
	mon.Wait()

	// Verify the mount transitioned to unhealthy
	if mount.GetStatus() != health.StatusUnhealthy {
		t.Errorf("expected mount status Unhealthy after %d failures, got %v", debounceThreshold, mount.GetStatus())
	}
}

func TestMonitor_DetectsRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	canaryPath := filepath.Join(tmpDir, ".health-check")

	// Don't create canary initially
	mount := health.NewMount(tmpDir, ".health-check")
	checker := health.NewChecker(100 * time.Millisecond)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	mon := monitor.New([]*health.Mount{mount}, checker, 50*time.Millisecond, 2, logger)

	ctx, cancel := context.WithCancel(context.Background())
	mon.Start(ctx)

	// Wait for failures to register (need at least 2 check intervals for debounce)
	time.Sleep(200 * time.Millisecond)

	// Now create the canary file to simulate recovery
	if err := os.WriteFile(canaryPath, []byte("ok"), 0644); err != nil {
		t.Fatalf("failed to create canary file: %v", err)
	}

	// Wait for recovery check (need at least 2 check intervals to be safe)
	time.Sleep(200 * time.Millisecond)

	cancel()
	mon.Wait()

	// Verify the mount recovered to healthy
	if mount.GetStatus() != health.StatusHealthy {
		t.Errorf("expected mount status Healthy after recovery, got %v", mount.GetStatus())
	}
}

func TestMonitor_MultipleMount(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Mount1: healthy
	canary1 := filepath.Join(tmpDir1, ".health-check")
	if err := os.WriteFile(canary1, []byte("ok"), 0644); err != nil {
		t.Fatalf("failed to create canary file: %v", err)
	}

	// Mount2: no canary file (unhealthy)
	mount1 := health.NewMount(tmpDir1, ".health-check")
	mount2 := health.NewMount(tmpDir2, ".health-check")

	checker := health.NewChecker(100 * time.Millisecond)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	debounceThreshold := 2
	mounts := []*health.Mount{mount1, mount2}
	mon := monitor.New(mounts, checker, 30*time.Millisecond, debounceThreshold, logger)

	ctx, cancel := context.WithCancel(context.Background())
	mon.Start(ctx)

	// Wait for enough checks to exceed debounce threshold for mount2
	// Need at least debounceThreshold+1 intervals (initial check + threshold failures)
	time.Sleep(time.Duration(debounceThreshold+2) * 50 * time.Millisecond)

	cancel()
	mon.Wait()

	// Mount1 should be healthy
	if mount1.GetStatus() != health.StatusHealthy {
		t.Errorf("expected mount1 status Healthy, got %v", mount1.GetStatus())
	}

	// Mount2 should be unhealthy (at least degraded due to no canary)
	status2 := mount2.GetStatus()
	if status2 != health.StatusUnhealthy && status2 != health.StatusDegraded {
		t.Errorf("expected mount2 status Unhealthy or Degraded, got %v", status2)
	}
}
