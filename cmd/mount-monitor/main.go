// Package main is the entry point for the mount health monitor service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/chris/debrid-mount-monitor/internal/config"
	"github.com/chris/debrid-mount-monitor/internal/health"
	"github.com/chris/debrid-mount-monitor/internal/monitor"
	"github.com/chris/debrid-mount-monitor/internal/server"
)

// Version is set at build time via ldflags.
var Version = "dev"

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	// Setup structured logging
	logger := setupLogger(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(logger)

	logger.Info("starting mount monitor",
		"version", Version,
		"mount_paths", cfg.MountPaths,
		"check_interval", cfg.CheckInterval.String(),
		"debounce_threshold", cfg.DebounceThreshold,
		"http_port", cfg.HTTPPort,
	)

	// Create mounts from configuration
	mounts := make([]*health.Mount, len(cfg.MountPaths))
	for i, path := range cfg.MountPaths {
		mounts[i] = health.NewMount(path, cfg.CanaryFile)
		logger.Info("mount registered",
			"path", path,
			"canary", mounts[i].CanaryPath,
		)
	}

	// Create health checker
	checker := health.NewChecker(cfg.ReadTimeout)

	// Create monitor
	mon := monitor.New(mounts, checker, cfg.CheckInterval, cfg.DebounceThreshold, logger)

	// Create HTTP server
	srv := server.New(mounts, cfg.HTTPPort, logger)

	// Setup shutdown context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start components
	mon.Start(ctx)
	if err := srv.Start(); err != nil {
		logger.Error("failed to start HTTP server", "error", err)
		os.Exit(1)
	}

	logger.Info("http server started", "port", cfg.HTTPPort)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	sig := <-sigChan
	logger.Info("received shutdown signal", "signal", sig.String())

	// Initiate graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	// Cancel monitor context to stop health checks
	cancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown error", "error", err)
	}

	// Wait for monitor to finish
	mon.Wait()

	logger.Info("shutdown complete")
	os.Exit(0)
}

// setupLogger creates a structured logger based on configuration.
func setupLogger(level, format string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
