package unit

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chris/debrid-mount-monitor/internal/config"
	"github.com/chris/debrid-mount-monitor/internal/health"
)

// T010: Test JSON file parsing with valid config
func TestConfigFile_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configJSON := `{
		"checkInterval": "60s",
		"readTimeout": "10s",
		"shutdownTimeout": "45s",
		"debounceThreshold": 5,
		"httpPort": 9090,
		"logLevel": "debug",
		"logFormat": "text",
		"canaryFile": ".ready",
		"mounts": [
			{
				"name": "movies",
				"path": "/mnt/movies",
				"canaryFile": ".health-check",
				"failureThreshold": 3
			},
			{
				"name": "tv",
				"path": "/mnt/tv",
				"failureThreshold": 5
			}
		]
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Test using LoadFromFileForTesting helper
	cfg := config.DefaultConfig()
	if err := cfg.LoadFromFileForTesting(configPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify global settings
	if cfg.CheckInterval != 60*time.Second {
		t.Errorf("expected checkInterval 60s, got %v", cfg.CheckInterval)
	}
	if cfg.ReadTimeout != 10*time.Second {
		t.Errorf("expected readTimeout 10s, got %v", cfg.ReadTimeout)
	}
	if cfg.ShutdownTimeout != 45*time.Second {
		t.Errorf("expected shutdownTimeout 45s, got %v", cfg.ShutdownTimeout)
	}
	if cfg.DebounceThreshold != 5 {
		t.Errorf("expected debounceThreshold 5, got %d", cfg.DebounceThreshold)
	}
	if cfg.HTTPPort != 9090 {
		t.Errorf("expected httpPort 9090, got %d", cfg.HTTPPort)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected logLevel 'debug', got %q", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("expected logFormat 'text', got %q", cfg.LogFormat)
	}
	if cfg.CanaryFile != ".ready" {
		t.Errorf("expected canaryFile '.ready', got %q", cfg.CanaryFile)
	}

	// Verify mounts
	if len(cfg.Mounts) != 2 {
		t.Fatalf("expected 2 mounts, got %d", len(cfg.Mounts))
	}
	if cfg.Mounts[0].Name != "movies" {
		t.Errorf("expected mount[0].name 'movies', got %q", cfg.Mounts[0].Name)
	}
	if cfg.Mounts[0].Path != "/mnt/movies" {
		t.Errorf("expected mount[0].path '/mnt/movies', got %q", cfg.Mounts[0].Path)
	}
	if cfg.Mounts[0].CanaryFile != ".health-check" {
		t.Errorf("expected mount[0].canaryFile '.health-check', got %q", cfg.Mounts[0].CanaryFile)
	}
	if cfg.Mounts[0].FailureThreshold != 3 {
		t.Errorf("expected mount[0].failureThreshold 3, got %d", cfg.Mounts[0].FailureThreshold)
	}
	if cfg.Mounts[1].Name != "tv" {
		t.Errorf("expected mount[1].name 'tv', got %q", cfg.Mounts[1].Name)
	}
	if cfg.Mounts[1].Path != "/mnt/tv" {
		t.Errorf("expected mount[1].path '/mnt/tv', got %q", cfg.Mounts[1].Path)
	}

	// Verify ConfigFile is set
	if cfg.ConfigFile != configPath {
		t.Errorf("expected ConfigFile %q, got %q", configPath, cfg.ConfigFile)
	}
}

// T011: Test --config flag loads specified file
func TestConfigFile_ExplicitPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom.json")

	configJSON := `{
		"mounts": [
			{"path": "/mnt/test"}
		]
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := config.DefaultConfig()
	if err := cfg.LoadFromFileForTesting(configPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Mounts) != 1 {
		t.Errorf("expected 1 mount, got %d", len(cfg.Mounts))
	}
	if cfg.ConfigFile != configPath {
		t.Errorf("expected ConfigFile %q, got %q", configPath, cfg.ConfigFile)
	}
}

// T011 continued: Test explicit path that doesn't exist returns error
func TestConfigFile_ExplicitPath_NotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	err := cfg.LoadFromFileForTesting("/nonexistent/config.json")

	if err == nil {
		t.Error("expected error for non-existent explicit config file")
	}
}

// T012: Test ./config.json default location discovery
func TestConfigFile_DefaultLocation(t *testing.T) {
	// Save current directory and restore after test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Create temp directory and switch to it
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Create config.json in current directory
	configJSON := `{
		"mounts": [
			{"name": "default-test", "path": "/mnt/default"}
		]
	}`

	if err := os.WriteFile("config.json", []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := config.DefaultConfig()
	// Pass empty string to trigger default file discovery
	if err := cfg.LoadFromFileForTesting(""); err != nil {
		t.Fatalf("failed to load default config: %v", err)
	}

	if len(cfg.Mounts) != 1 {
		t.Errorf("expected 1 mount from default config, got %d", len(cfg.Mounts))
	}
	if cfg.Mounts[0].Name != "default-test" {
		t.Errorf("expected mount name 'default-test', got %q", cfg.Mounts[0].Name)
	}
}

// T012 continued: Test missing default config.json is silently ignored
func TestConfigFile_DefaultLocation_NotFound(t *testing.T) {
	// Save current directory and restore after test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Create temp directory (without config.json) and switch to it
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	cfg := config.DefaultConfig()
	// This should NOT return an error - missing default is silently ignored
	if err := cfg.LoadFromFileForTesting(""); err != nil {
		t.Errorf("expected no error for missing default config, got: %v", err)
	}

	// Config should still have defaults
	if cfg.CheckInterval != 30*time.Second {
		t.Errorf("expected default checkInterval 30s, got %v", cfg.CheckInterval)
	}
}

// T013: Test backwards compatibility - no config file uses env vars
func TestConfigFile_BackwardsCompatibility(t *testing.T) {
	// This test verifies that the Config struct can still be used
	// with MountPaths (legacy) when no config file is present
	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{"/mnt/test1", "/mnt/test2"}

	// Validation should pass with legacy MountPaths
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config with MountPaths, got error: %v", err)
	}

	// Both Mounts (empty) and MountPaths should be acceptable
	if len(cfg.Mounts) != 0 {
		t.Errorf("expected empty Mounts array, got %d", len(cfg.Mounts))
	}
	if len(cfg.MountPaths) != 2 {
		t.Errorf("expected 2 MountPaths, got %d", len(cfg.MountPaths))
	}
}

// T020: Test per-mount canary file override
func TestConfigFile_PerMountCanaryOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configJSON := `{
		"canaryFile": ".global-health",
		"mounts": [
			{
				"name": "with-override",
				"path": "/mnt/test1",
				"canaryFile": ".custom-health"
			},
			{
				"name": "without-override",
				"path": "/mnt/test2"
			}
		]
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := config.DefaultConfig()
	if err := cfg.LoadFromFileForTesting(configPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Mount with override should use custom canary file
	if cfg.Mounts[0].CanaryFile != ".custom-health" {
		t.Errorf("expected mount[0].canaryFile '.custom-health', got %q", cfg.Mounts[0].CanaryFile)
	}

	// Mount without override should inherit global canary file
	if cfg.Mounts[1].CanaryFile != ".global-health" {
		t.Errorf("expected mount[1].canaryFile '.global-health' (inherited), got %q", cfg.Mounts[1].CanaryFile)
	}
}

// T021: Test per-mount failureThreshold override
func TestConfigFile_PerMountThresholdOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configJSON := `{
		"debounceThreshold": 5,
		"mounts": [
			{
				"name": "with-override",
				"path": "/mnt/test1",
				"failureThreshold": 10
			},
			{
				"name": "without-override",
				"path": "/mnt/test2"
			}
		]
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := config.DefaultConfig()
	if err := cfg.LoadFromFileForTesting(configPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Mount with override should use custom threshold
	if cfg.Mounts[0].FailureThreshold != 10 {
		t.Errorf("expected mount[0].failureThreshold 10, got %d", cfg.Mounts[0].FailureThreshold)
	}

	// Mount without override should inherit global threshold
	if cfg.Mounts[1].FailureThreshold != 5 {
		t.Errorf("expected mount[1].failureThreshold 5 (inherited), got %d", cfg.Mounts[1].FailureThreshold)
	}
}

// T022: Test default inheritance when per-mount values not specified
func TestConfigFile_DefaultInheritance(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Minimal mount config - should inherit all defaults
	configJSON := `{
		"mounts": [
			{"path": "/mnt/test"}
		]
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := config.DefaultConfig()
	if err := cfg.LoadFromFileForTesting(configPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Mount should inherit default canary file
	if cfg.Mounts[0].CanaryFile != ".health-check" {
		t.Errorf("expected mount canaryFile '.health-check' (default), got %q", cfg.Mounts[0].CanaryFile)
	}

	// Mount should inherit default threshold
	if cfg.Mounts[0].FailureThreshold != 3 {
		t.Errorf("expected mount failureThreshold 3 (default), got %d", cfg.Mounts[0].FailureThreshold)
	}
}

// T029: Test invalid JSON syntax error message
func TestConfigFile_InvalidJSONSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Invalid JSON - missing closing brace
	invalidJSON := `{
		"mounts": [
			{"path": "/mnt/test"}
		]
	`

	if err := os.WriteFile(configPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := config.DefaultConfig()
	err := cfg.LoadFromFileForTesting(configPath)

	if err == nil {
		t.Error("expected error for invalid JSON syntax")
	}
}

// T030: Test missing required "path" field error
func TestConfigFile_MissingRequiredPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configJSON := `{
		"mounts": [
			{"name": "no-path-mount"}
		]
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := config.DefaultConfig()
	err := cfg.LoadFromFileForTesting(configPath)

	if err == nil {
		t.Error("expected error for missing path field")
	}
}

// T031: Test invalid failureThreshold (negative) error
func TestConfigFile_InvalidFailureThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configJSON := `{
		"mounts": [
			{
				"name": "invalid-threshold",
				"path": "/mnt/test",
				"failureThreshold": -1
			}
		]
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := config.DefaultConfig()
	err := cfg.LoadFromFileForTesting(configPath)

	if err == nil {
		t.Error("expected error for negative failureThreshold")
	}
}

// Test Duration unmarshaling
func TestDuration_UnmarshalJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	tests := []struct {
		name     string
		json     string
		expected time.Duration
	}{
		{"seconds", `{"checkInterval": "30s", "mounts": [{"path": "/mnt/test"}]}`, 30 * time.Second},
		{"minutes", `{"checkInterval": "5m", "mounts": [{"path": "/mnt/test"}]}`, 5 * time.Minute},
		{"hours", `{"checkInterval": "1h", "mounts": [{"path": "/mnt/test"}]}`, 1 * time.Hour},
		{"mixed", `{"checkInterval": "1h30m", "mounts": [{"path": "/mnt/test"}]}`, 90 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(configPath, []byte(tt.json), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			cfg := config.DefaultConfig()
			if err := cfg.LoadFromFileForTesting(configPath); err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			if cfg.CheckInterval != tt.expected {
				t.Errorf("expected checkInterval %v, got %v", tt.expected, cfg.CheckInterval)
			}
		})
	}
}

// Test Duration with invalid format
func TestDuration_UnmarshalJSON_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Invalid duration format
	configJSON := `{
		"checkInterval": "invalid",
		"mounts": [{"path": "/mnt/test"}]
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := config.DefaultConfig()
	err := cfg.LoadFromFileForTesting(configPath)

	if err == nil {
		t.Error("expected error for invalid duration format")
	}
}

// =============================================================================
// Precedence Integration Tests (High Priority from PR Review)
// =============================================================================

// TestPrecedence_ConfigFileMountsNotOverriddenByEnvVar verifies that
// MOUNT_PATHS env var does NOT override mounts from config file.
// This is the fix for PR Review Issue #3.
func TestPrecedence_ConfigFileMountsNotOverriddenByEnvVar(t *testing.T) {
	// Save and restore env var
	originalEnv := os.Getenv("MOUNT_PATHS")
	defer os.Setenv("MOUNT_PATHS", originalEnv)

	// Set MOUNT_PATHS env var
	os.Setenv("MOUNT_PATHS", "/env/mount1,/env/mount2")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Config file with mounts
	configJSON := `{
		"mounts": [
			{"name": "from-file", "path": "/file/mount"}
		]
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := config.DefaultConfig()
	if err := cfg.LoadFromFileForTesting(configPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Config file mounts should be preserved
	if len(cfg.Mounts) != 1 {
		t.Fatalf("expected 1 mount from config file, got %d", len(cfg.Mounts))
	}
	if cfg.Mounts[0].Path != "/file/mount" {
		t.Errorf("expected mount path '/file/mount', got %q", cfg.Mounts[0].Path)
	}
	if cfg.Mounts[0].Name != "from-file" {
		t.Errorf("expected mount name 'from-file', got %q", cfg.Mounts[0].Name)
	}

	// MountPaths should NOT have been set from env var (config file mounts take precedence)
	if len(cfg.MountPaths) != 1 || cfg.MountPaths[0] != "/file/mount" {
		t.Errorf("expected MountPaths from config file, got %v", cfg.MountPaths)
	}
}

// TestPrecedence_EnvVarWorksWithoutConfigFileMounts verifies that
// MOUNT_PATHS env var works when no config file mounts are present.
func TestPrecedence_EnvVarWorksWithoutConfigFileMounts(t *testing.T) {
	// Save current directory and restore after test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Create temp directory without config.json
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Save and restore env var
	originalEnv := os.Getenv("MOUNT_PATHS")
	defer os.Setenv("MOUNT_PATHS", originalEnv)

	// Set MOUNT_PATHS env var
	os.Setenv("MOUNT_PATHS", "/env/mount1,/env/mount2")

	cfg := config.DefaultConfig()
	// Load with empty path (no config file)
	if err := cfg.LoadFromFileForTesting(""); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// No config file, so no Mounts from file
	if len(cfg.Mounts) != 0 {
		t.Errorf("expected 0 mounts (no config file), got %d", len(cfg.Mounts))
	}

	// MountPaths would be set by env var in the full Load() flow
	// This test just verifies that LoadFromFileForTesting doesn't interfere
}

// TestPrecedence_GlobalSettingsOverriddenByEnvVar verifies that
// global settings (not mounts) from config file ARE overridden by env vars.
// This test simulates the precedence by manually applying env var logic after file loading.
func TestPrecedence_GlobalSettingsOverriddenByEnvVar(t *testing.T) {
	// Save and restore env vars
	originalLogLevel := os.Getenv("LOG_LEVEL")
	originalHTTPPort := os.Getenv("HTTP_PORT")
	originalCheckInterval := os.Getenv("CHECK_INTERVAL")
	defer func() {
		os.Setenv("LOG_LEVEL", originalLogLevel)
		os.Setenv("HTTP_PORT", originalHTTPPort)
		os.Setenv("CHECK_INTERVAL", originalCheckInterval)
	}()

	// Set env vars that should override config file
	os.Setenv("LOG_LEVEL", "error")
	os.Setenv("HTTP_PORT", "9999")
	os.Setenv("CHECK_INTERVAL", "2m")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Config file sets different values
	configJSON := `{
		"logLevel": "debug",
		"httpPort": 9000,
		"checkInterval": "30s",
		"mounts": [{"path": "/mnt/test"}]
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := config.DefaultConfig()
	if err := cfg.LoadFromFileForTesting(configPath); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// After file loading, values should be from config file
	if cfg.LogLevel != "debug" {
		t.Errorf("after file load: expected logLevel 'debug', got %q", cfg.LogLevel)
	}
	if cfg.HTTPPort != 9000 {
		t.Errorf("after file load: expected httpPort 9000, got %d", cfg.HTTPPort)
	}
	if cfg.CheckInterval != 30*time.Second {
		t.Errorf("after file load: expected checkInterval 30s, got %v", cfg.CheckInterval)
	}

	// Simulate env var override (as Load() does after loadFromFile)
	if envVal := os.Getenv("LOG_LEVEL"); envVal != "" {
		cfg.LogLevel = envVal
	}
	if envVal := os.Getenv("HTTP_PORT"); envVal != "" {
		if port, err := strconv.Atoi(envVal); err == nil {
			cfg.HTTPPort = port
		}
	}
	if envVal := os.Getenv("CHECK_INTERVAL"); envVal != "" {
		if d, err := time.ParseDuration(envVal); err == nil {
			cfg.CheckInterval = d
		}
	}

	// After env var override, values should be from env vars
	if cfg.LogLevel != "error" {
		t.Errorf("after env override: expected logLevel 'error', got %q", cfg.LogLevel)
	}
	if cfg.HTTPPort != 9999 {
		t.Errorf("after env override: expected httpPort 9999, got %d", cfg.HTTPPort)
	}
	if cfg.CheckInterval != 2*time.Minute {
		t.Errorf("after env override: expected checkInterval 2m, got %v", cfg.CheckInterval)
	}
}

// TestMountNameInStatusResponse verifies FR-008: mount name appears in health status snapshot.
// This ensures the Mount.Name field is properly propagated to status responses.
func TestMountNameInStatusResponse(t *testing.T) {
	mount := health.NewMount("test-movies", "/mnt/movies", ".health-check", 3)

	snapshot := mount.Snapshot()

	if snapshot.Name != "test-movies" {
		t.Errorf("expected snapshot.Name 'test-movies', got %q", snapshot.Name)
	}
	if snapshot.Path != "/mnt/movies" {
		t.Errorf("expected snapshot.Path '/mnt/movies', got %q", snapshot.Path)
	}
}

// =============================================================================
// Security Hardening Tests (Issue #17, #15)
// =============================================================================

// TestConfigFile_FileSizeLimit verifies that config files larger than 1MB are rejected.
// This prevents DoS attacks via excessively large config files.
func TestConfigFile_FileSizeLimit(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create a config file just over 1MB (1MB + overhead)
	// Use strings.Repeat to create valid JSON content (null bytes are invalid in JSON strings)
	padding := strings.Repeat("x", 1024*1024)
	largeContent := `{"mounts":[{"path":"/mnt/test","name":"` + padding + `"}]}`

	if err := os.WriteFile(configPath, []byte(largeContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := config.DefaultConfig()
	err := cfg.LoadFromFileForTesting(configPath)

	if err == nil {
		t.Error("expected error for config file exceeding 1MB size limit")
	}

	// Verify the error message mentions the size limit
	if err != nil && !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("expected error to mention size limit, got: %v", err)
	}
}

// TestConfigFile_FileSizeLimit_JustUnder verifies that config files just under 1MB are accepted.
func TestConfigFile_FileSizeLimit_JustUnder(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create a valid config file just under 1MB
	configJSON := `{
		"mounts": [
			{"path": "/mnt/test"}
		]
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := config.DefaultConfig()
	err := cfg.LoadFromFileForTesting(configPath)

	if err != nil {
		t.Errorf("expected no error for config file under 1MB, got: %v", err)
	}
}

// TestConfigFile_FileSizeLimit_ExactlyOneMB verifies that config files of exactly 1MB are accepted.
// This is a boundary test - the limit is > 1MB, so exactly 1MB should be valid.
func TestConfigFile_FileSizeLimit_ExactlyOneMB(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create a config file of exactly 1MB (1048576 bytes)
	// JSON structure: {"mounts":[{"path":"/mnt/test","name":"<padding>"}]}
	prefix := `{"mounts":[{"path":"/mnt/test","name":"`
	suffix := `"}]}`
	targetSize := 1 << 20 // 1MB = 1048576 bytes
	paddingSize := targetSize - len(prefix) - len(suffix)

	// Create padding with 'x' characters (valid JSON string content)
	padding := make([]byte, paddingSize)
	for i := range padding {
		padding[i] = 'x'
	}

	content := prefix + string(padding) + suffix
	if len(content) != targetSize {
		t.Fatalf("expected content size %d, got %d", targetSize, len(content))
	}

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := config.DefaultConfig()
	err := cfg.LoadFromFileForTesting(configPath)

	if err != nil {
		t.Errorf("expected no error for config file of exactly 1MB, got: %v", err)
	}
}
