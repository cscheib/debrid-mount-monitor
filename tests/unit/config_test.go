package unit

import (
	"testing"
	"time"

	"github.com/cscheib/debrid-mount-monitor/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg.CanaryFile != ".health-check" {
		t.Errorf("expected default canary file '.health-check', got %q", cfg.CanaryFile)
	}
	if cfg.CheckInterval != 30*time.Second {
		t.Errorf("expected default check interval 30s, got %v", cfg.CheckInterval)
	}
	if cfg.ReadTimeout != 5*time.Second {
		t.Errorf("expected default read timeout 5s, got %v", cfg.ReadTimeout)
	}
	if cfg.FailureThreshold != 3 {
		t.Errorf("expected default failure threshold 3, got %d", cfg.FailureThreshold)
	}
	if cfg.HTTPPort != 8080 {
		t.Errorf("expected default HTTP port 8080, got %d", cfg.HTTPPort)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default log level 'info', got %q", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("expected default log format 'json', got %q", cfg.LogFormat)
	}
}

func TestConfigValidation_Valid(t *testing.T) {
	cfg := &config.Config{
		MountPaths:       []string{"/mnt/test"},
		CanaryFile:       ".health-check",
		CheckInterval:    30 * time.Second,
		ReadTimeout:      5 * time.Second,
		ShutdownTimeout:  30 * time.Second,
		FailureThreshold: 3,
		HTTPPort:         8080,
		LogLevel:         "info",
		LogFormat:        "json",
		Watchdog: config.WatchdogConfig{
			Enabled:             false,
			RestartDelay:        0,
			MaxRetries:          3,
			RetryBackoffInitial: 100 * time.Millisecond,
			RetryBackoffMax:     10 * time.Second,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}
}

func TestConfigValidation_NoMountPaths(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for empty mount paths")
	}
}

func TestConfigValidation_CheckIntervalTooShort(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{"/mnt/test"}
	cfg.CheckInterval = 500 * time.Millisecond

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for check interval < 1s")
	}
}

func TestConfigValidation_ReadTimeoutTooShort(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{"/mnt/test"}
	cfg.ReadTimeout = 50 * time.Millisecond

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for read timeout < 100ms")
	}
}

func TestConfigValidation_ReadTimeoutExceedsCheckInterval(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{"/mnt/test"}
	cfg.CheckInterval = 5 * time.Second
	cfg.ReadTimeout = 10 * time.Second

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for read timeout >= check interval")
	}
}

func TestConfigValidation_InvalidFailureThreshold(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{"/mnt/test"}
	cfg.FailureThreshold = 0

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for failure threshold < 1")
	}
}

func TestConfigValidation_InvalidHTTPPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero", 0},
		{"negative", -1},
		{"too high", 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.MountPaths = []string{"/mnt/test"}
			cfg.HTTPPort = tt.port

			err := cfg.Validate()
			if err == nil {
				t.Errorf("expected validation error for HTTP port %d", tt.port)
			}
		})
	}
}

func TestConfigValidation_InvalidLogLevel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{"/mnt/test"}
	cfg.LogLevel = "invalid"

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for invalid log level")
	}
}

func TestConfigValidation_InvalidLogFormat(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{"/mnt/test"}
	cfg.LogFormat = "xml"

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for invalid log format")
	}
}

func TestConfigValidation_AllLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}
	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.MountPaths = []string{"/mnt/test"}
			cfg.LogLevel = level

			if err := cfg.Validate(); err != nil {
				t.Errorf("expected log level %q to be valid, got error: %v", level, err)
			}
		})
	}
}

func TestConfigValidation_AllLogFormats(t *testing.T) {
	formats := []string{"json", "text"}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.MountPaths = []string{"/mnt/test"}
			cfg.LogFormat = format

			if err := cfg.Validate(); err != nil {
				t.Errorf("expected log format %q to be valid, got error: %v", format, err)
			}
		})
	}
}
