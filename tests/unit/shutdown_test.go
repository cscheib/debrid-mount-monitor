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
	"github.com/matryer/is"
)

func TestGracefulShutdown_ServerStops(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("", "/tmp", ".nonexistent", 3)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Use a high port to avoid conflicts
	srv := server.New([]*health.Mount{mount}, 18082, "test", logger)

	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	is.NoErr(srv.Shutdown(ctx)) // shutdown should succeed
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
	is := is.New(t)

	mount := health.NewMount("", "/tmp", ".nonexistent", 3)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	srv := server.New([]*health.Mount{mount}, 18081, "test", logger)

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
		is.Equal(resp.StatusCode, http.StatusOK) // server should respond with 200
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	is.NoErr(srv.Shutdown(ctx)) // shutdown should succeed

	// Verify server is stopped
	_, err = http.Get("http://localhost:18081/healthz/live")
	is.True(err != nil) // connection should fail after shutdown
}
