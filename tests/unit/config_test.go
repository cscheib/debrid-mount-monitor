package unit

import (
	"testing"
	"time"

	"github.com/cscheib/debrid-mount-monitor/internal/config"
	"github.com/matryer/is"
)

func TestDefaultConfig(t *testing.T) {
	is := is.New(t)
	cfg := config.DefaultConfig()

	is.Equal(cfg.CanaryFile, ".health-check")   // default canary file
	is.Equal(cfg.CheckInterval, 30*time.Second) // default check interval
	is.Equal(cfg.ReadTimeout, 5*time.Second)    // default read timeout
	is.Equal(cfg.FailureThreshold, 3)           // default failure threshold
	is.Equal(cfg.HTTPPort, 8080)                // default HTTP port
	is.Equal(cfg.LogLevel, "info")              // default log level
	is.Equal(cfg.LogFormat, "json")             // default log format
}

func TestConfigValidation_Valid(t *testing.T) {
	is := is.New(t)

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

	is.NoErr(cfg.Validate()) // valid config should not error
}

func TestConfigValidation_NoMountPaths(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{}

	err := cfg.Validate()
	is.True(err != nil) // empty mount paths should error
}

func TestConfigValidation_CheckIntervalTooShort(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{"/mnt/test"}
	cfg.CheckInterval = 500 * time.Millisecond

	err := cfg.Validate()
	is.True(err != nil) // check interval < 1s should error
}

func TestConfigValidation_ReadTimeoutTooShort(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{"/mnt/test"}
	cfg.ReadTimeout = 50 * time.Millisecond

	err := cfg.Validate()
	is.True(err != nil) // read timeout < 100ms should error
}

func TestConfigValidation_ReadTimeoutExceedsCheckInterval(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{"/mnt/test"}
	cfg.CheckInterval = 5 * time.Second
	cfg.ReadTimeout = 10 * time.Second

	err := cfg.Validate()
	is.True(err != nil) // read timeout >= check interval should error
}

func TestConfigValidation_InvalidFailureThreshold(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{"/mnt/test"}
	cfg.FailureThreshold = 0

	err := cfg.Validate()
	is.True(err != nil) // failure threshold < 1 should error
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
			is := is.New(t)

			cfg := config.DefaultConfig()
			cfg.MountPaths = []string{"/mnt/test"}
			cfg.HTTPPort = tt.port

			err := cfg.Validate()
			is.True(err != nil) // invalid HTTP port should error
		})
	}
}

func TestConfigValidation_InvalidLogLevel(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{"/mnt/test"}
	cfg.LogLevel = "invalid"

	err := cfg.Validate()
	is.True(err != nil) // invalid log level should error
}

func TestConfigValidation_InvalidLogFormat(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{"/mnt/test"}
	cfg.LogFormat = "xml"

	err := cfg.Validate()
	is.True(err != nil) // invalid log format should error
}

func TestConfigValidation_AllLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}
	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			is := is.New(t)

			cfg := config.DefaultConfig()
			cfg.MountPaths = []string{"/mnt/test"}
			cfg.LogLevel = level

			is.NoErr(cfg.Validate()) // valid log level should not error
		})
	}
}

func TestConfigValidation_AllLogFormats(t *testing.T) {
	formats := []string{"json", "text"}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			is := is.New(t)

			cfg := config.DefaultConfig()
			cfg.MountPaths = []string{"/mnt/test"}
			cfg.LogFormat = format

			is.NoErr(cfg.Validate()) // valid log format should not error
		})
	}
}
