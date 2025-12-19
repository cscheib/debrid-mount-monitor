package config_test

import (
	"testing"
	"time"

	"github.com/cscheib/debrid-mount-monitor/internal/config"
	"github.com/matryer/is"
	"go.uber.org/goleak"
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
	defer goleak.VerifyNone(t)
	is := is.New(t)

	cfg := &config.Config{
		Mounts: []config.MountConfig{
			{Path: "/mnt/test", CanaryFile: ".health-check", FailureThreshold: 3},
		},
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

// testMount returns a simple mount config for testing
func testMount() config.MountConfig {
	return config.MountConfig{Path: "/mnt/test", CanaryFile: ".health-check", FailureThreshold: 3}
}

func TestConfigValidation_NoMounts(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.Mounts = []config.MountConfig{}

	err := cfg.Validate()
	is.True(err != nil) // empty mounts should error
}

func TestConfigValidation_CheckIntervalTooShort(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.Mounts = []config.MountConfig{testMount()}
	cfg.CheckInterval = 500 * time.Millisecond

	err := cfg.Validate()
	is.True(err != nil) // check interval < 1s should error
}

func TestConfigValidation_ReadTimeoutTooShort(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.Mounts = []config.MountConfig{testMount()}
	cfg.ReadTimeout = 50 * time.Millisecond

	err := cfg.Validate()
	is.True(err != nil) // read timeout < 100ms should error
}

func TestConfigValidation_ReadTimeoutExceedsCheckInterval(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.Mounts = []config.MountConfig{testMount()}
	cfg.CheckInterval = 5 * time.Second
	cfg.ReadTimeout = 10 * time.Second

	err := cfg.Validate()
	is.True(err != nil) // read timeout >= check interval should error
}

func TestConfigValidation_InvalidFailureThreshold(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.Mounts = []config.MountConfig{testMount()}
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
			cfg.Mounts = []config.MountConfig{testMount()}
			cfg.HTTPPort = tt.port

			err := cfg.Validate()
			is.True(err != nil) // invalid HTTP port should error
		})
	}
}

func TestConfigValidation_InvalidLogLevel(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.Mounts = []config.MountConfig{testMount()}
	cfg.LogLevel = "invalid"

	err := cfg.Validate()
	is.True(err != nil) // invalid log level should error
}

func TestConfigValidation_InvalidLogFormat(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.Mounts = []config.MountConfig{testMount()}
	cfg.LogFormat = "xml"

	err := cfg.Validate()
	is.True(err != nil) // invalid log format should error
}

func TestConfigValidation_ShutdownTimeoutTooShort(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.Mounts = []config.MountConfig{testMount()}
	cfg.ShutdownTimeout = 500 * time.Millisecond

	err := cfg.Validate()
	is.True(err != nil) // shutdown timeout < 1s should error
}

func TestConfigValidation_AllLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}
	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			is := is.New(t)

			cfg := config.DefaultConfig()
			cfg.Mounts = []config.MountConfig{testMount()}
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
			cfg.Mounts = []config.MountConfig{testMount()}
			cfg.LogFormat = format

			is.NoErr(cfg.Validate()) // valid log format should not error
		})
	}
}

// TestConfigValidation_MountMissingPath tests mount without path
func TestConfigValidation_MountMissingPath(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.Mounts = []config.MountConfig{{Path: ""}} // Mount with no path

	err := cfg.Validate()
	is.True(err != nil) // mount without path should error
}

// TestConfigValidation_MountWithNameMissingPath tests named mount without path
func TestConfigValidation_MountWithNameMissingPath(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.Mounts = []config.MountConfig{{Name: "my-mount", Path: ""}} // Named mount with no path

	err := cfg.Validate()
	is.True(err != nil) // named mount without path should error
}

// TestConfigValidation_MountNegativeThreshold tests mount with negative failure threshold
func TestConfigValidation_MountNegativeThreshold(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.Mounts = []config.MountConfig{{Path: "/mnt/test", FailureThreshold: -1}}

	err := cfg.Validate()
	is.True(err != nil) // negative failure threshold should error
}

// TestConfigValidation_NamedMountNegativeThreshold tests named mount with negative failure threshold
func TestConfigValidation_NamedMountNegativeThreshold(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.Mounts = []config.MountConfig{{Name: "my-mount", Path: "/mnt/test", FailureThreshold: -1}}

	err := cfg.Validate()
	is.True(err != nil) // named mount with negative failure threshold should error
}

// TestConfigValidation_MountEmptyCanaryFile tests that mount with empty canary inherits global default
func TestConfigValidation_MountEmptyCanaryFile(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.Mounts = []config.MountConfig{{Path: "/mnt/test", CanaryFile: ""}} // Empty canary - should use global

	err := cfg.Validate()
	is.NoErr(err) // empty canary file should be valid (inherits global default)
}

// TestConfigValidation_MountZeroThreshold tests that mount with threshold=0 is valid (uses global default)
func TestConfigValidation_MountZeroThreshold(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	cfg.Mounts = []config.MountConfig{{Path: "/mnt/test", FailureThreshold: 0}} // Zero = use global

	err := cfg.Validate()
	is.NoErr(err) // threshold=0 should be valid (sentinel for "use global default")
}

// T004: Test that InitContainerMode field exists and defaults to false
func TestDefaultConfig_InitContainerMode(t *testing.T) {
	is := is.New(t)
	cfg := config.DefaultConfig()

	is.Equal(cfg.InitContainerMode, false) // init-container mode defaults to false
}

// T005: Test that init-container mode skips irrelevant validations
func TestValidate_InitContainerMode_SkipsIrrelevant(t *testing.T) {
	is := is.New(t)

	// Create a config with invalid values for fields that should be skipped
	cfg := &config.Config{
		InitContainerMode: true,
		Mounts: []config.MountConfig{
			{Path: "/mnt/test", CanaryFile: ".health-check"},
		},
		CanaryFile:  ".health-check",
		ReadTimeout: 5 * time.Second, // Valid - this is still checked
		LogLevel:    "info",          // Valid - this is still checked
		LogFormat:   "json",          // Valid - this is still checked
		// These should be skipped in init-container mode:
		CheckInterval:    0, // Invalid in normal mode
		ShutdownTimeout:  0, // Invalid in normal mode
		FailureThreshold: 0, // Invalid in normal mode
		HTTPPort:         0, // Invalid in normal mode
		Watchdog: config.WatchdogConfig{
			MaxRetries:          0, // Invalid in normal mode
			RetryBackoffInitial: 0, // Invalid in normal mode
			RetryBackoffMax:     0, // Invalid in normal mode
		},
	}

	err := cfg.Validate()
	is.NoErr(err) // init-container mode should skip irrelevant validations
}

// Test that init-container mode still validates required fields
func TestValidate_InitContainerMode_StillValidatesRequired(t *testing.T) {
	is := is.New(t)

	// Config with no mounts - should still fail in init-container mode
	cfg := &config.Config{
		InitContainerMode: true,
		Mounts:            []config.MountConfig{},
		ReadTimeout:       5 * time.Second,
		LogLevel:          "info",
		LogFormat:         "json",
	}

	err := cfg.Validate()
	is.True(err != nil) // at least one mount is required even in init-container mode
}

// Test that init-container mode validates ReadTimeout (it's used for canary checks)
func TestValidate_InitContainerMode_ValidatesReadTimeout(t *testing.T) {
	is := is.New(t)

	cfg := &config.Config{
		InitContainerMode: true,
		Mounts: []config.MountConfig{
			{Path: "/mnt/test"},
		},
		ReadTimeout: 10 * time.Millisecond, // Too short - should fail
		LogLevel:    "info",
		LogFormat:   "json",
	}

	err := cfg.Validate()
	is.True(err != nil) // read timeout validation should still apply
}
