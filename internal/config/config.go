// Package config handles configuration parsing from JSON files and CLI flags.
package config

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	flag "github.com/spf13/pflag"
)

// MountConfig holds per-mount configuration settings.
type MountConfig struct {
	Name             string // Human-readable identifier (optional)
	Path             string // Filesystem path to mount point (required) - can be absolute or relative
	CanaryFile       string // Relative path to canary file within mount (optional, inherits global)
	FailureThreshold int    // Consecutive failures before unhealthy (0 = use global failureThreshold)
}

// WatchdogConfig holds configuration for the watchdog feature.
type WatchdogConfig struct {
	Enabled             bool          // Enable/disable watchdog mode (default: false)
	RestartDelay        time.Duration // Delay after UNHEALTHY before triggering restart (default: 0s)
	MaxRetries          int           // Max API retry attempts before fallback (default: 3)
	RetryBackoffInitial time.Duration // Initial retry delay for exponential backoff (default: 100ms)
	RetryBackoffMax     time.Duration // Maximum retry delay cap (default: 10s)
}

// Config holds all runtime configuration for the mount monitor.
type Config struct {
	// Config file tracking
	ConfigFile string // Path to loaded config file ("" if none)

	// Mount configuration
	Mounts     []MountConfig // Per-mount configurations
	CanaryFile string        // Default canary file for all mounts

	// Timing configuration
	CheckInterval   time.Duration // Time between health checks
	ReadTimeout     time.Duration // Timeout for canary file read
	ShutdownTimeout time.Duration // Max time for graceful shutdown

	// Failure threshold configuration
	FailureThreshold int // Default consecutive failures before unhealthy

	// Server configuration
	HTTPPort int // Port for health endpoints

	// Logging configuration
	LogLevel  string // debug, info, warn, error
	LogFormat string // json, text

	// Watchdog configuration
	Watchdog WatchdogConfig // Pod restart watchdog settings
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		ConfigFile:       "",
		Mounts:           []MountConfig{},
		CanaryFile:       ".health-check",
		CheckInterval:    30 * time.Second,
		ReadTimeout:      5 * time.Second,
		ShutdownTimeout:  30 * time.Second,
		FailureThreshold: 3,
		HTTPPort:         8080,
		LogLevel:         "info",
		LogFormat:        "json",
		Watchdog: WatchdogConfig{
			Enabled:             false,
			RestartDelay:        0,
			MaxRetries:          3,
			RetryBackoffInitial: 100 * time.Millisecond,
			RetryBackoffMax:     10 * time.Second,
		},
	}
}

// Load parses configuration from config file and command-line flags.
// Precedence: Defaults → Config File → CLI Flags
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Define flags
	// Note: Most configuration is done via JSON config file. Only essential runtime flags are kept.
	configFile := flag.StringP("config", "c", "", "Path to JSON configuration file")
	httpPort := flag.Int("http-port", 0, "Port for health endpoints")
	logLevel := flag.String("log-level", "", "Log level: debug, info, warn, error")
	logFormat := flag.String("log-format", "", "Log format: json, text")

	flag.Parse()

	// Load from config file (after defaults, before CLI flags)
	if err := cfg.loadFromFile(*configFile); err != nil {
		return nil, err
	}

	// Override with flags if provided (only runtime essentials)
	if *httpPort > 0 {
		cfg.HTTPPort = *httpPort
	}
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}
	if *logFormat != "" {
		cfg.LogFormat = *logFormat
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	var result *multierror.Error

	// Check for mounts
	if len(c.Mounts) == 0 {
		result = multierror.Append(result, fmt.Errorf("at least one mount is required"))
	}

	// Validate individual mount configs
	for i, m := range c.Mounts {
		if m.Path == "" {
			if m.Name != "" {
				result = multierror.Append(result, fmt.Errorf("mount[%d] %q: path is required", i, m.Name))
			} else {
				result = multierror.Append(result, fmt.Errorf("mount[%d]: path is required", i))
			}
		}
		if m.FailureThreshold < 0 {
			if m.Name != "" {
				result = multierror.Append(result, fmt.Errorf("mount[%d] %q: failureThreshold must be >= 0", i, m.Name))
			} else {
				result = multierror.Append(result, fmt.Errorf("mount[%d]: failureThreshold must be >= 0", i))
			}
		}
	}

	if c.CheckInterval < time.Second {
		result = multierror.Append(result, fmt.Errorf("check interval must be >= 1 second"))
	}

	if c.ReadTimeout < 100*time.Millisecond {
		result = multierror.Append(result, fmt.Errorf("read timeout must be >= 100 milliseconds"))
	}

	if c.ReadTimeout >= c.CheckInterval {
		result = multierror.Append(result, fmt.Errorf("read timeout must be less than check interval (otherwise health checks would overlap or never complete before the next check)"))
	}

	if c.ShutdownTimeout < time.Second {
		result = multierror.Append(result, fmt.Errorf("shutdown timeout must be >= 1 second"))
	}

	if c.FailureThreshold < 1 {
		result = multierror.Append(result, fmt.Errorf("failure threshold must be >= 1"))
	}

	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		result = multierror.Append(result, fmt.Errorf("HTTP port must be between 1 and 65535"))
	}

	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.LogLevel] {
		result = multierror.Append(result, fmt.Errorf("log level must be one of: debug, info, warn, error (got %q)", c.LogLevel))
	}

	validLogFormats := map[string]bool{"json": true, "text": true}
	if !validLogFormats[c.LogFormat] {
		result = multierror.Append(result, fmt.Errorf("log format must be one of: json, text (got %q)", c.LogFormat))
	}

	// Validate watchdog configuration
	if c.Watchdog.RestartDelay < 0 {
		result = multierror.Append(result, fmt.Errorf("watchdog restart delay must be >= 0"))
	}
	if c.Watchdog.MaxRetries < 1 {
		result = multierror.Append(result, fmt.Errorf("watchdog max retries must be >= 1"))
	}
	if c.Watchdog.RetryBackoffInitial <= 0 {
		result = multierror.Append(result, fmt.Errorf("watchdog retry backoff initial must be > 0"))
	}
	if c.Watchdog.RetryBackoffMax < c.Watchdog.RetryBackoffInitial {
		result = multierror.Append(result, fmt.Errorf("watchdog retry backoff max must be >= retry backoff initial"))
	}

	return result.ErrorOrNil()
}
