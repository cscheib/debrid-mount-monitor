// Package config handles configuration parsing from files, environment variables, and flags.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
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

// FileConfig represents the JSON configuration file structure.
type FileConfig struct {
	CheckInterval     Duration          `json:"checkInterval,omitempty"`
	ReadTimeout       Duration          `json:"readTimeout,omitempty"`
	ShutdownTimeout   Duration          `json:"shutdownTimeout,omitempty"`
	DebounceThreshold int               `json:"debounceThreshold,omitempty"`
	HTTPPort          int               `json:"httpPort,omitempty"`
	LogLevel          string            `json:"logLevel,omitempty"`
	LogFormat         string            `json:"logFormat,omitempty"`
	CanaryFile        string            `json:"canaryFile,omitempty"`
	Mounts            []FileMountConfig `json:"mounts,omitempty"`
}

// FileMountConfig represents per-mount configuration in the JSON file.
type FileMountConfig struct {
	Name             string `json:"name,omitempty"`
	Path             string `json:"path"`
	CanaryFile       string `json:"canaryFile,omitempty"`
	FailureThreshold int    `json:"failureThreshold,omitempty"`
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

	// Check if file exists
	_, err := os.Stat(filePath)
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

// LoadFromFileForTesting is a test helper that exposes loadFromFile for testing.
// It should only be used in tests.
func (c *Config) LoadFromFileForTesting(configPath string) error {
	return c.loadFromFile(configPath)
}

// applyFileConfig applies values from FileConfig to the runtime Config.
// Values from the file override defaults but will be overridden by env vars/flags.
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
	if fc.DebounceThreshold > 0 {
		c.DebounceThreshold = fc.DebounceThreshold
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
				mc.FailureThreshold = c.DebounceThreshold
			}

			c.Mounts[i] = mc
			c.MountPaths[i] = fm.Path
		}
	}
}
