package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cscheib/debrid-mount-monitor/internal/config"
	"github.com/matryer/is"
)

func TestSetupLogger_Levels(t *testing.T) {
	tests := []struct {
		level    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo}, // default
		{"", slog.LevelInfo},        // default
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			is := is.New(t)

			logger := setupLogger(tt.level, "json")

			is.True(logger != nil) // logger should be created
		})
	}
}

func TestSetupLogger_Formats(t *testing.T) {
	tests := []struct {
		format string
	}{
		{"json"},
		{"text"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			is := is.New(t)

			logger := setupLogger("info", tt.format)

			is.True(logger != nil) // logger should be created
		})
	}
}

func TestMultiStreamHandler_Enabled(t *testing.T) {
	is := is.New(t)

	handler := &multiStreamHandler{
		level:         slog.LevelInfo,
		stdoutHandler: slog.NewJSONHandler(os.Stdout, nil),
		stderrHandler: slog.NewJSONHandler(os.Stderr, nil),
	}

	// Info and above should be enabled when level is Info
	is.True(handler.Enabled(context.Background(), slog.LevelInfo))
	is.True(handler.Enabled(context.Background(), slog.LevelWarn))
	is.True(handler.Enabled(context.Background(), slog.LevelError))

	// Debug should not be enabled when level is Info
	is.True(!handler.Enabled(context.Background(), slog.LevelDebug))
}

func TestMultiStreamHandler_Handle_Routing(t *testing.T) {
	is := is.New(t)

	var stdoutBuf, stderrBuf bytes.Buffer

	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	handler := &multiStreamHandler{
		level:         slog.LevelDebug,
		stdoutHandler: slog.NewTextHandler(&stdoutBuf, opts),
		stderrHandler: slog.NewTextHandler(&stderrBuf, opts),
	}

	logger := slog.New(handler)

	// Debug and Info should go to stdout
	logger.Debug("debug message")
	logger.Info("info message")

	// Warn and Error should go to stderr
	logger.Warn("warn message")
	logger.Error("error message")

	// Verify stdout got debug/info
	stdoutContent := stdoutBuf.String()
	is.True(strings.Contains(stdoutContent, "debug message"))  // stdout should have debug
	is.True(strings.Contains(stdoutContent, "info message"))   // stdout should have info
	is.True(!strings.Contains(stdoutContent, "warn message"))  // stdout should not have warn
	is.True(!strings.Contains(stdoutContent, "error message")) // stdout should not have error

	// Verify stderr got warn/error
	stderrContent := stderrBuf.String()
	is.True(!strings.Contains(stderrContent, "debug message")) // stderr should not have debug
	is.True(!strings.Contains(stderrContent, "info message"))  // stderr should not have info
	is.True(strings.Contains(stderrContent, "warn message"))   // stderr should have warn
	is.True(strings.Contains(stderrContent, "error message"))  // stderr should have error
}

func TestMultiStreamHandler_WithAttrs(t *testing.T) {
	is := is.New(t)

	var buf bytes.Buffer
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	handler := &multiStreamHandler{
		level:         slog.LevelInfo,
		stdoutHandler: slog.NewTextHandler(&buf, opts),
		stderrHandler: slog.NewTextHandler(&buf, opts),
	}

	// WithAttrs should return a new handler
	newHandler := handler.WithAttrs([]slog.Attr{slog.String("key", "value")})

	is.True(newHandler != nil)     // should return handler
	is.True(newHandler != handler) // should be a new handler
}

func TestMultiStreamHandler_WithGroup(t *testing.T) {
	is := is.New(t)

	var buf bytes.Buffer
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	handler := &multiStreamHandler{
		level:         slog.LevelInfo,
		stdoutHandler: slog.NewTextHandler(&buf, opts),
		stderrHandler: slog.NewTextHandler(&buf, opts),
	}

	// WithGroup should return a new handler
	newHandler := handler.WithGroup("mygroup")

	is.True(newHandler != nil)     // should return handler
	is.True(newHandler != handler) // should be a new handler
}

func TestRunInitMode_AllHealthy(t *testing.T) {
	is := is.New(t)

	// Create temp directory with canary file
	tmpDir := t.TempDir()
	canaryPath := filepath.Join(tmpDir, ".health-check")
	err := os.WriteFile(canaryPath, []byte("ok"), 0644)
	is.NoErr(err)

	cfg := &config.Config{
		InitContainerMode: true,
		ReadTimeout:       time.Second,
		Mounts: []config.MountConfig{
			{
				Name:             "test-mount",
				Path:             tmpDir,
				CanaryFile:       ".health-check",
				FailureThreshold: 3,
			},
		},
	}

	// Create silent logger for test
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	exitCode := runInitMode(cfg, logger)

	is.Equal(exitCode, 0) // should return 0 when all mounts are healthy
}

func TestRunInitMode_SomeUnhealthy(t *testing.T) {
	is := is.New(t)

	// Create temp directory WITHOUT canary file
	tmpDir := t.TempDir()

	cfg := &config.Config{
		InitContainerMode: true,
		ReadTimeout:       time.Second,
		Mounts: []config.MountConfig{
			{
				Name:             "unhealthy-mount",
				Path:             tmpDir,
				CanaryFile:       ".health-check",
				FailureThreshold: 3,
			},
		},
	}

	// Create silent logger for test
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	exitCode := runInitMode(cfg, logger)

	is.Equal(exitCode, 1) // should return 1 when any mount is unhealthy
}

func TestRunInitMode_MultipleMounts_MixedHealth(t *testing.T) {
	is := is.New(t)

	// Create healthy mount
	healthyDir := t.TempDir()
	err := os.WriteFile(filepath.Join(healthyDir, ".health-check"), []byte("ok"), 0644)
	is.NoErr(err)

	// Create unhealthy mount (no canary)
	unhealthyDir := t.TempDir()

	cfg := &config.Config{
		InitContainerMode: true,
		ReadTimeout:       time.Second,
		Mounts: []config.MountConfig{
			{
				Name:             "healthy",
				Path:             healthyDir,
				CanaryFile:       ".health-check",
				FailureThreshold: 3,
			},
			{
				Name:             "unhealthy",
				Path:             unhealthyDir,
				CanaryFile:       ".health-check",
				FailureThreshold: 3,
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	exitCode := runInitMode(cfg, logger)

	is.Equal(exitCode, 1) // should return 1 if ANY mount is unhealthy
}

func TestRunInitMode_EmptyMounts(t *testing.T) {
	is := is.New(t)

	cfg := &config.Config{
		InitContainerMode: true,
		ReadTimeout:       time.Second,
		Mounts:            []config.MountConfig{},
	}

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	exitCode := runInitMode(cfg, logger)

	is.Equal(exitCode, 0) // no mounts = all healthy (vacuously true)
}
