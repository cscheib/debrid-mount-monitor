// Package main is the entry point for the mount health monitor service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/cscheib/debrid-mount-monitor/internal/config"
	"github.com/cscheib/debrid-mount-monitor/internal/health"
	"github.com/cscheib/debrid-mount-monitor/internal/monitor"
	"github.com/cscheib/debrid-mount-monitor/internal/server"
	"github.com/cscheib/debrid-mount-monitor/internal/watchdog"
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

	// Log configuration source and settings (T037: verbose config logging)
	configSource := "environment"
	if cfg.ConfigFile != "" {
		configSource = cfg.ConfigFile
	}

	logger.Info("configuration loaded",
		"source", configSource,
		"mounts", len(cfg.Mounts),
	)

	logger.Info("starting mount monitor",
		"version", Version,
		"check_interval", cfg.CheckInterval.String(),
		"read_timeout", cfg.ReadTimeout.String(),
		"shutdown_timeout", cfg.ShutdownTimeout.String(),
		"failure_threshold", cfg.FailureThreshold,
		"http_port", cfg.HTTPPort,
		"log_level", cfg.LogLevel,
		"log_format", cfg.LogFormat,
		"canary_file", cfg.CanaryFile,
		"watchdog_enabled", cfg.Watchdog.Enabled,
		"watchdog_restart_delay", cfg.Watchdog.RestartDelay.String(),
	)

	// Create mounts from configuration
	mounts := make([]*health.Mount, len(cfg.Mounts))
	for i, mc := range cfg.Mounts {
		mounts[i] = health.NewMount(mc.Name, mc.Path, mc.CanaryFile, mc.FailureThreshold)
		logger.Info("mount registered",
			"name", mc.Name,
			"path", mc.Path,
			"canary", mounts[i].CanaryPath,
			"failureThreshold", mc.FailureThreshold,
		)
	}

	// Create health checker
	checker := health.NewChecker(cfg.ReadTimeout)

	// Create monitor
	mon := monitor.New(mounts, checker, cfg.CheckInterval, cfg.FailureThreshold, logger)

	// Initialize watchdog
	// Read pod identity from Downward API environment variables
	podName := os.Getenv("POD_NAME")
	podNamespace := os.Getenv("POD_NAMESPACE")

	// Warn if watchdog is enabled but pod identity is not configured
	if cfg.Watchdog.Enabled && (podName == "" || podNamespace == "") {
		logger.Warn("watchdog enabled but pod identity not configured",
			"pod_name_set", podName != "",
			"pod_namespace_set", podNamespace != "",
			"hint", "set POD_NAME and POD_NAMESPACE env vars via Downward API")
	}

	watchdogCfg := watchdog.Config{
		Enabled:             cfg.Watchdog.Enabled,
		RestartDelay:        cfg.Watchdog.RestartDelay,
		MaxRetries:          cfg.Watchdog.MaxRetries,
		RetryBackoffInitial: cfg.Watchdog.RetryBackoffInitial,
		RetryBackoffMax:     cfg.Watchdog.RetryBackoffMax,
	}
	wd := watchdog.NewWatchdog(watchdogCfg, podName, podNamespace, logger)

	// Connect watchdog to monitor for state change notifications
	mon.SetWatchdog(wd)

	// Create HTTP server
	srv := server.New(mounts, cfg.HTTPPort, Version, logger)

	// Setup shutdown context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watchdog (validates K8s access and RBAC)
	if err := wd.Start(ctx); err != nil {
		logger.Error("watchdog start failed", "error", err)
		// Non-fatal - continue without watchdog
	}

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
// Per FR-012: debug/info → stdout, warn/error → stderr
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

	opts := &slog.HandlerOptions{Level: logLevel}

	// Pre-create handlers for stdout and stderr to avoid allocation on every log call
	var stdoutHandler, stderrHandler slog.Handler
	if format == "text" {
		stdoutHandler = slog.NewTextHandler(os.Stdout, opts)
		stderrHandler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		stdoutHandler = slog.NewJSONHandler(os.Stdout, opts)
		stderrHandler = slog.NewJSONHandler(os.Stderr, opts)
	}

	// Create a multi-stream handler that routes by log level
	handler := &multiStreamHandler{
		level:         logLevel,
		stdoutHandler: stdoutHandler,
		stderrHandler: stderrHandler,
	}

	return slog.New(handler)
}

// multiStreamHandler routes logs to stdout or stderr based on level.
// debug, info → stdout; warn, error → stderr
// Handlers are pre-created to avoid allocation on every log call.
type multiStreamHandler struct {
	level         slog.Level
	stdoutHandler slog.Handler
	stderrHandler slog.Handler
}

func (h *multiStreamHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *multiStreamHandler) Handle(ctx context.Context, r slog.Record) error {
	// Route to appropriate pre-created handler based on level
	if r.Level >= slog.LevelWarn {
		return h.stderrHandler.Handle(ctx, r)
	}
	return h.stdoutHandler.Handle(ctx, r)
}

func (h *multiStreamHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &multiStreamHandler{
		level:         h.level,
		stdoutHandler: h.stdoutHandler.WithAttrs(attrs),
		stderrHandler: h.stderrHandler.WithAttrs(attrs),
	}
}

func (h *multiStreamHandler) WithGroup(name string) slog.Handler {
	return &multiStreamHandler{
		level:         h.level,
		stdoutHandler: h.stdoutHandler.WithGroup(name),
		stderrHandler: h.stderrHandler.WithGroup(name),
	}
}
