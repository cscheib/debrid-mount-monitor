package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cscheib/debrid-mount-monitor/internal/health"
	"github.com/matryer/is"
)

func TestChecker_HealthyMount(t *testing.T) {
	is := is.New(t)

	// Create temporary directory with canary file
	tmpDir := t.TempDir()
	canaryPath := filepath.Join(tmpDir, ".health-check")
	if err := os.WriteFile(canaryPath, []byte("ok"), 0644); err != nil {
		t.Fatalf("failed to create canary file: %v", err)
	}

	mount := health.NewMount("", tmpDir, ".health-check", 3)
	checker := health.NewChecker(5 * time.Second)

	result := checker.Check(context.Background(), mount)

	is.True(result.Success)      // check should succeed
	is.NoErr(result.Error)       // no error expected
	is.True(result.Duration > 0) // duration should be positive
}

func TestChecker_MissingCanaryFile(t *testing.T) {
	is := is.New(t)

	tmpDir := t.TempDir()
	// Don't create canary file

	mount := health.NewMount("", tmpDir, ".health-check", 3)
	checker := health.NewChecker(5 * time.Second)

	result := checker.Check(context.Background(), mount)

	is.True(!result.Success)     // check should fail for missing canary file
	is.True(result.Error != nil) // error expected for missing canary file
}

func TestChecker_MissingMountPath(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/nonexistent/path/that/does/not/exist", ".health-check", 3)
	checker := health.NewChecker(5 * time.Second)

	result := checker.Check(context.Background(), mount)

	is.True(!result.Success)     // check should fail for missing mount path
	is.True(result.Error != nil) // error expected for missing mount path
}

func TestChecker_Timeout(t *testing.T) {
	is := is.New(t)

	// This test verifies the timeout context is respected
	// We can't easily simulate a hanging filesystem, but we can test the context

	tmpDir := t.TempDir()
	canaryPath := filepath.Join(tmpDir, ".health-check")
	if err := os.WriteFile(canaryPath, []byte("ok"), 0644); err != nil {
		t.Fatalf("failed to create canary file: %v", err)
	}

	mount := health.NewMount("", tmpDir, ".health-check", 3)
	checker := health.NewChecker(100 * time.Millisecond)

	// Use an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := checker.Check(ctx, mount)

	// The check should fail due to cancelled context
	is.True(!result.Success) // check should fail with cancelled context
}

func TestChecker_PermissionDenied(t *testing.T) {
	is := is.New(t)

	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	canaryPath := filepath.Join(tmpDir, ".health-check")
	if err := os.WriteFile(canaryPath, []byte("ok"), 0000); err != nil {
		t.Fatalf("failed to create canary file: %v", err)
	}
	defer os.Chmod(canaryPath, 0644) // Cleanup

	mount := health.NewMount("", tmpDir, ".health-check", 3)
	checker := health.NewChecker(5 * time.Second)

	result := checker.Check(context.Background(), mount)

	is.True(!result.Success)     // check should fail due to permission denied
	is.True(result.Error != nil) // permission error expected
}

func TestChecker_EmptyCanaryFile(t *testing.T) {
	is := is.New(t)

	// Empty canary file should still be considered healthy
	tmpDir := t.TempDir()
	canaryPath := filepath.Join(tmpDir, ".health-check")
	if err := os.WriteFile(canaryPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create canary file: %v", err)
	}

	mount := health.NewMount("", tmpDir, ".health-check", 3)
	checker := health.NewChecker(5 * time.Second)

	result := checker.Check(context.Background(), mount)

	is.True(result.Success) // empty canary file should be healthy
}
