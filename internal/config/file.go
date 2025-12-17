// Package config handles configuration parsing from JSON files and CLI flags.
package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"
)

const (
	// maxConfigFileSize is the maximum allowed config file size (1MB).
	// This prevents DoS attacks via excessively large config files.
	maxConfigFileSize = 1 << 20 // 1MB

	// worldWritableBits is the Unix permission bit for "other write" access.
	// Used to detect world-writable config files which are a security risk.
	worldWritableBits = 0002
)

// Duration is a wrapper around time.Duration that supports JSON unmarshaling from strings.
type Duration time.Duration

// UnmarshalJSON implements json.Unmarshaler for Duration.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

// FileWatchdogConfig represents watchdog configuration in the JSON file.
type FileWatchdogConfig struct {
	Enabled             *bool    `json:"enabled,omitempty"`
	RestartDelay        Duration `json:"restartDelay,omitempty"`
	MaxRetries          int      `json:"maxRetries,omitempty"`
	RetryBackoffInitial Duration `json:"retryBackoffInitial,omitempty"`
	RetryBackoffMax     Duration `json:"retryBackoffMax,omitempty"`
}

// FileConfig represents the JSON configuration file structure.
type FileConfig struct {
	CheckInterval    Duration           `json:"checkInterval,omitempty"`
	ReadTimeout      Duration           `json:"readTimeout,omitempty"`
	ShutdownTimeout  Duration           `json:"shutdownTimeout,omitempty"`
	FailureThreshold int                `json:"failureThreshold,omitempty"`
	HTTPPort         int                `json:"httpPort,omitempty"`
	LogLevel         string             `json:"logLevel,omitempty"`
	LogFormat        string             `json:"logFormat,omitempty"`
	CanaryFile       string             `json:"canaryFile,omitempty"`
	Mounts           []FileMountConfig  `json:"mounts,omitempty"`
	Watchdog         FileWatchdogConfig `json:"watchdog,omitempty"`
}

// FileMountConfig represents per-mount configuration in the JSON file.
type FileMountConfig struct {
	Name             string `json:"name,omitempty"`
	Path             string `json:"path"`
	CanaryFile       string `json:"canaryFile,omitempty"`
	FailureThreshold int    `json:"failureThreshold,omitempty"` // 0 = use global default, >= 1 = explicit value
}

// defaultConfigPath is the default location to check for a config file.
const defaultConfigPath = "./config.json"

// loadFromFile loads configuration from a JSON file.
// If configPath is empty, it checks for ./config.json as a default.
// If the default doesn't exist, it silently continues (backwards compatible).
// If an explicit path is provided but doesn't exist, it returns an error.
func (c *Config) loadFromFile(configPath string) error {
	var filePath string
	var explicitPath bool

	if configPath != "" {
		// Explicit path provided via --config flag
		filePath = configPath
		explicitPath = true
	} else {
		// Check for default config file
		filePath = defaultConfigPath
		explicitPath = false
	}

	// Check if file exists and get file info
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		if explicitPath {
			return fmt.Errorf("config file not found: %s", filePath)
		}
		// Default config file doesn't exist - silently continue (backwards compatible)
		return nil
	}
	if err != nil {
		return fmt.Errorf("error checking config file: %w", err)
	}

	// Check file size to prevent DoS via excessively large files
	if info.Size() > maxConfigFileSize {
		return fmt.Errorf("config file %s exceeds maximum size of %d bytes (got %d bytes)",
			filePath, maxConfigFileSize, info.Size())
	}

	// Warn if config file is world-writable (security risk) - Unix only
	// Note: This warning uses the default slog logger since it runs before setupLogger()
	// in main(). The warning will use Go's default text format, which is acceptable for
	// startup security warnings.
	if runtime.GOOS != "windows" {
		if info.Mode().Perm()&worldWritableBits != 0 {
			slog.Warn("config file is world-writable, which may be a security risk",
				"path", filePath,
				"mode", fmt.Sprintf("%04o", info.Mode().Perm()))
		}
	}

	// Read and parse the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading config file %s: %w", filePath, err)
	}

	var fileConfig FileConfig
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("error parsing config file %s: %w", filePath, err)
	}

	// Validate the file config before applying
	if err := validateFileConfig(&fileConfig); err != nil {
		return fmt.Errorf("config file %s: %w", filePath, err)
	}

	// Apply file config to runtime config
	c.ConfigFile = filePath
	applyFileConfig(c, &fileConfig)

	return nil
}

// validateFileConfig checks that the file configuration is valid.
func validateFileConfig(fc *FileConfig) error {
	// If mounts are specified, each must have a path
	for i, m := range fc.Mounts {
		if m.Path == "" {
			if m.Name != "" {
				return fmt.Errorf("mount[%d] %q: missing required field \"path\"", i, m.Name)
			}
			return fmt.Errorf("mount[%d]: missing required field \"path\"", i)
		}
		// FailureThreshold of 0 means "use default", but negative is invalid
		if m.FailureThreshold < 0 {
			if m.Name != "" {
				return fmt.Errorf("mount[%d] %q: failureThreshold must be >= 0, got %d", i, m.Name, m.FailureThreshold)
			}
			return fmt.Errorf("mount[%d]: failureThreshold must be >= 0, got %d", i, m.FailureThreshold)
		}
	}

	return nil
}

// applyFileConfig applies values from FileConfig to the runtime Config.
// Values from the file override defaults but will be overridden by CLI flags.
func applyFileConfig(c *Config, fc *FileConfig) {
	// Apply global settings if specified
	if fc.CheckInterval > 0 {
		c.CheckInterval = time.Duration(fc.CheckInterval)
	}
	if fc.ReadTimeout > 0 {
		c.ReadTimeout = time.Duration(fc.ReadTimeout)
	}
	if fc.ShutdownTimeout > 0 {
		c.ShutdownTimeout = time.Duration(fc.ShutdownTimeout)
	}
	if fc.FailureThreshold > 0 {
		c.FailureThreshold = fc.FailureThreshold
	}
	if fc.HTTPPort > 0 {
		c.HTTPPort = fc.HTTPPort
	}
	if fc.LogLevel != "" {
		c.LogLevel = fc.LogLevel
	}
	if fc.LogFormat != "" {
		c.LogFormat = fc.LogFormat
	}
	if fc.CanaryFile != "" {
		c.CanaryFile = fc.CanaryFile
	}

	// Apply mount configurations with inheritance
	if len(fc.Mounts) > 0 {
		c.Mounts = make([]MountConfig, len(fc.Mounts))
		c.MountPaths = make([]string, len(fc.Mounts))

		for i, fm := range fc.Mounts {
			// Apply per-mount config with inheritance from globals
			mc := MountConfig{
				Name: fm.Name,
				Path: fm.Path,
			}

			// Inherit canary file from global if not specified
			if fm.CanaryFile != "" {
				mc.CanaryFile = fm.CanaryFile
			} else {
				mc.CanaryFile = c.CanaryFile
			}

			// Inherit failure threshold from global if not specified (0 means use default)
			if fm.FailureThreshold > 0 {
				mc.FailureThreshold = fm.FailureThreshold
			} else {
				mc.FailureThreshold = c.FailureThreshold
			}

			c.Mounts[i] = mc
			c.MountPaths[i] = fm.Path
		}
	}

	// Apply watchdog configuration
	// Note: Using pointer for Enabled to distinguish "not set" from "set to false"
	if fc.Watchdog.Enabled != nil {
		c.Watchdog.Enabled = *fc.Watchdog.Enabled
	}
	if fc.Watchdog.RestartDelay > 0 {
		c.Watchdog.RestartDelay = time.Duration(fc.Watchdog.RestartDelay)
	}
	if fc.Watchdog.MaxRetries > 0 {
		c.Watchdog.MaxRetries = fc.Watchdog.MaxRetries
	}
	if fc.Watchdog.RetryBackoffInitial > 0 {
		c.Watchdog.RetryBackoffInitial = time.Duration(fc.Watchdog.RetryBackoffInitial)
	}
	if fc.Watchdog.RetryBackoffMax > 0 {
		c.Watchdog.RetryBackoffMax = time.Duration(fc.Watchdog.RetryBackoffMax)
	}
}
