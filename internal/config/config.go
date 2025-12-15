// Package config handles configuration parsing from environment variables and flags.
package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// MountConfig holds per-mount configuration settings.
type MountConfig struct {
	Name             string // Human-readable identifier (optional)
	Path             string // Filesystem path to mount point (required)
	CanaryFile       string // Relative path to canary file (optional, inherits global)
	FailureThreshold int    // Consecutive failures before unhealthy (0 = use global debounceThreshold)
}

// Config holds all runtime configuration for the mount monitor.
type Config struct {
	// Config file tracking
	ConfigFile string // Path to loaded config file ("" if none)

	// Mount configuration
	MountPaths []string      // Paths to monitor (legacy, derived from Mounts)
	Mounts     []MountConfig // Per-mount configurations
	CanaryFile string        // Default canary file for all mounts

	// Timing configuration
	CheckInterval   time.Duration // Time between health checks
	ReadTimeout     time.Duration // Timeout for canary file read
	ShutdownTimeout time.Duration // Max time for graceful shutdown

	// Debounce configuration
	DebounceThreshold int // Default consecutive failures before unhealthy

	// Server configuration
	HTTPPort int // Port for health endpoints

	// Logging configuration
	LogLevel  string // debug, info, warn, error
	LogFormat string // json, text
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		ConfigFile:        "",
		MountPaths:        []string{},
		Mounts:            []MountConfig{},
		CanaryFile:        ".health-check",
		CheckInterval:     30 * time.Second,
		ReadTimeout:       5 * time.Second,
		ShutdownTimeout:   30 * time.Second,
		DebounceThreshold: 3,
		HTTPPort:          8080,
		LogLevel:          "info",
		LogFormat:         "json",
	}
}

// Load parses configuration from config file, environment variables, and command-line flags.
// Precedence: Defaults → Config File → Environment Variables → CLI Flags
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Define flags
	configFile := flag.String("config", "", "Path to JSON configuration file")
	flag.StringVar(configFile, "c", "", "Path to JSON configuration file (shorthand)")
	mountPaths := flag.String("mount-paths", "", "Comma-separated list of mount paths to monitor")
	canaryFile := flag.String("canary-file", "", "Relative path to canary file within each mount")
	checkInterval := flag.Duration("check-interval", 0, "Time between health checks")
	readTimeout := flag.Duration("read-timeout", 0, "Timeout for canary file read")
	shutdownTimeout := flag.Duration("shutdown-timeout", 0, "Max time for graceful shutdown")
	debounceThreshold := flag.Int("debounce-threshold", 0, "Consecutive failures before unhealthy")
	httpPort := flag.Int("http-port", 0, "Port for health endpoints")
	logLevel := flag.String("log-level", "", "Log level: debug, info, warn, error")
	logFormat := flag.String("log-format", "", "Log format: json, text")

	flag.Parse()

	// Load from config file (after defaults, before env vars)
	if err := cfg.loadFromFile(*configFile); err != nil {
		return nil, err
	}

	// Load from environment variables
	// Note: MOUNT_PATHS only applies if no mounts were loaded from config file
	// This preserves per-mount config from file while allowing env vars for legacy setups
	if envVal := os.Getenv("MOUNT_PATHS"); envVal != "" && len(cfg.Mounts) == 0 {
		cfg.MountPaths = parseMountPaths(envVal)
	}
	if envVal := os.Getenv("CANARY_FILE"); envVal != "" {
		cfg.CanaryFile = envVal
	}
	if envVal := os.Getenv("CHECK_INTERVAL"); envVal != "" {
		if d, err := time.ParseDuration(envVal); err == nil {
			cfg.CheckInterval = d
		}
	}
	if envVal := os.Getenv("READ_TIMEOUT"); envVal != "" {
		if d, err := time.ParseDuration(envVal); err == nil {
			cfg.ReadTimeout = d
		}
	}
	if envVal := os.Getenv("SHUTDOWN_TIMEOUT"); envVal != "" {
		if d, err := time.ParseDuration(envVal); err == nil {
			cfg.ShutdownTimeout = d
		}
	}
	if envVal := os.Getenv("DEBOUNCE_THRESHOLD"); envVal != "" {
		if i, err := strconv.Atoi(envVal); err == nil {
			cfg.DebounceThreshold = i
		}
	}
	if envVal := os.Getenv("HTTP_PORT"); envVal != "" {
		if i, err := strconv.Atoi(envVal); err == nil {
			cfg.HTTPPort = i
		}
	}
	if envVal := os.Getenv("LOG_LEVEL"); envVal != "" {
		cfg.LogLevel = envVal
	}
	if envVal := os.Getenv("LOG_FORMAT"); envVal != "" {
		cfg.LogFormat = envVal
	}

	// Override with flags if provided
	// Note: --mount-paths flag takes highest precedence and clears config file mounts
	if *mountPaths != "" {
		cfg.MountPaths = parseMountPaths(*mountPaths)
		cfg.Mounts = nil // Clear config file mounts - CLI flag takes precedence
	}
	if *canaryFile != "" {
		cfg.CanaryFile = *canaryFile
	}
	if *checkInterval > 0 {
		cfg.CheckInterval = *checkInterval
	}
	if *readTimeout > 0 {
		cfg.ReadTimeout = *readTimeout
	}
	if *shutdownTimeout > 0 {
		cfg.ShutdownTimeout = *shutdownTimeout
	}
	if *debounceThreshold > 0 {
		cfg.DebounceThreshold = *debounceThreshold
	}
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
	var errs []string

	// Check for mounts - either via Mounts or legacy MountPaths
	if len(c.Mounts) == 0 && len(c.MountPaths) == 0 {
		errs = append(errs, "at least one mount path is required")
	}

	// Validate individual mount configs
	for i, m := range c.Mounts {
		if m.Path == "" {
			if m.Name != "" {
				errs = append(errs, fmt.Sprintf("mount[%d] %q: path is required", i, m.Name))
			} else {
				errs = append(errs, fmt.Sprintf("mount[%d]: path is required", i))
			}
		}
		if m.FailureThreshold < 0 {
			if m.Name != "" {
				errs = append(errs, fmt.Sprintf("mount[%d] %q: failureThreshold must be >= 0", i, m.Name))
			} else {
				errs = append(errs, fmt.Sprintf("mount[%d]: failureThreshold must be >= 0", i))
			}
		}
	}

	if c.CheckInterval < time.Second {
		errs = append(errs, "check interval must be >= 1 second")
	}

	if c.ReadTimeout < 100*time.Millisecond {
		errs = append(errs, "read timeout must be >= 100 milliseconds")
	}

	if c.ReadTimeout >= c.CheckInterval {
		errs = append(errs, "read timeout must be less than check interval")
	}

	if c.DebounceThreshold < 1 {
		errs = append(errs, "debounce threshold must be >= 1")
	}

	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		errs = append(errs, "HTTP port must be between 1 and 65535")
	}

	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.LogLevel] {
		errs = append(errs, fmt.Sprintf("log level must be one of: debug, info, warn, error (got %q)", c.LogLevel))
	}

	validLogFormats := map[string]bool{"json": true, "text": true}
	if !validLogFormats[c.LogFormat] {
		errs = append(errs, fmt.Sprintf("log format must be one of: json, text (got %q)", c.LogFormat))
	}

	if len(errs) > 0 {
		return errors.New("configuration validation failed: " + strings.Join(errs, "; "))
	}

	return nil
}

// parseMountPaths splits a comma-separated string into a slice of paths.
func parseMountPaths(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	paths := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}
