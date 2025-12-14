package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chris/debrid-mount-monitor/internal/health"
)

func TestChecker_HealthyMount(t *testing.T) {
	// Create temporary directory with canary file
	tmpDir := t.TempDir()
	canaryPath := filepath.Join(tmpDir, ".health-check")
	if err := os.WriteFile(canaryPath, []byte("ok"), 0644); err != nil {
		t.Fatalf("failed to create canary file: %v", err)
	}

	mount := health.NewMount(tmpDir, ".health-check")
	checker := health.NewChecker(5 * time.Second)

	result := checker.Check(context.Background(), mount)

	if !result.Success {
		t.Errorf("expected check to succeed, got error: %v", result.Error)
	}
	if result.Error != nil {
		t.Errorf("expected no error, got: %v", result.Error)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

func TestChecker_MissingCanaryFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create canary file

	mount := health.NewMount(tmpDir, ".health-check")
	checker := health.NewChecker(5 * time.Second)

	result := checker.Check(context.Background(), mount)

	if result.Success {
		t.Error("expected check to fail for missing canary file")
	}
	if result.Error == nil {
		t.Error("expected error for missing canary file")
	}
}

func TestChecker_MissingMountPath(t *testing.T) {
	mount := health.NewMount("/nonexistent/path/that/does/not/exist", ".health-check")
	checker := health.NewChecker(5 * time.Second)

	result := checker.Check(context.Background(), mount)

	if result.Success {
		t.Error("expected check to fail for missing mount path")
	}
	if result.Error == nil {
		t.Error("expected error for missing mount path")
	}
}

func TestChecker_Timeout(t *testing.T) {
	// This test verifies the timeout context is respected
	// We can't easily simulate a hanging filesystem, but we can test the context

	tmpDir := t.TempDir()
	canaryPath := filepath.Join(tmpDir, ".health-check")
	if err := os.WriteFile(canaryPath, []byte("ok"), 0644); err != nil {
		t.Fatalf("failed to create canary file: %v", err)
	}

	mount := health.NewMount(tmpDir, ".health-check")
	checker := health.NewChecker(100 * time.Millisecond)

	// Use an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := checker.Check(ctx, mount)

	// The check should fail due to cancelled context
	if result.Success {
		t.Error("expected check to fail with cancelled context")
	}
}

func TestChecker_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	canaryPath := filepath.Join(tmpDir, ".health-check")
	if err := os.WriteFile(canaryPath, []byte("ok"), 0000); err != nil {
		t.Fatalf("failed to create canary file: %v", err)
	}
	defer os.Chmod(canaryPath, 0644) // Cleanup

	mount := health.NewMount(tmpDir, ".health-check")
	checker := health.NewChecker(5 * time.Second)

	result := checker.Check(context.Background(), mount)

	if result.Success {
		t.Error("expected check to fail due to permission denied")
	}
	if result.Error == nil {
		t.Error("expected permission error")
	}
}

func TestChecker_EmptyCanaryFile(t *testing.T) {
	// Empty canary file should still be considered healthy
	tmpDir := t.TempDir()
	canaryPath := filepath.Join(tmpDir, ".health-check")
	if err := os.WriteFile(canaryPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create canary file: %v", err)
	}

	mount := health.NewMount(tmpDir, ".health-check")
	checker := health.NewChecker(5 * time.Second)

	result := checker.Check(context.Background(), mount)

	if !result.Success {
		t.Errorf("expected empty canary file to be healthy, got error: %v", result.Error)
	}
}
