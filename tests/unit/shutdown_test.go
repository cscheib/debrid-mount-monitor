package unit

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cscheib/debrid-mount-monitor/internal/health"
	"github.com/cscheib/debrid-mount-monitor/internal/monitor"
	"github.com/cscheib/debrid-mount-monitor/internal/server"
)

func TestGracefulShutdown_ServerStops(t *testing.T) {
	mount := health.NewMount("", "/tmp", ".nonexistent", 3)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Use a high port to avoid conflicts
	srv := server.New([]*health.Mount{mount}, 18082, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("shutdown returned error: %v", err)
	}
}

func TestGracefulShutdown_MonitorStops(t *testing.T) {
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

	// Give it time to run
	time.Sleep(50 * time.Millisecond)

	// Simulate shutdown signal
	cancel()

	// Wait should complete without hanging
	done := make(chan struct{})
	go func() {
		mon.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - monitor stopped
	case <-time.After(2 * time.Second):
		t.Error("monitor did not stop within timeout")
	}
}

func TestGracefulShutdown_InFlightRequests(t *testing.T) {
	mount := health.NewMount("", "/tmp", ".nonexistent", 3)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	srv := server.New([]*health.Mount{mount}, 18081, logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Make a request to verify server is working
	resp, err := http.Get("http://localhost:18081/healthz/live")
	if err != nil {
		t.Logf("warning: could not verify server is running: %v", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("shutdown returned error: %v", err)
	}

	// Verify server is stopped
	_, err = http.Get("http://localhost:18081/healthz/live")
	if err == nil {
		t.Error("expected connection error after shutdown")
	}
}
