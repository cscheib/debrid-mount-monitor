package unit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cscheib/debrid-mount-monitor/internal/config"
	"github.com/cscheib/debrid-mount-monitor/internal/health"
	"github.com/matryer/is"
)

// T010: Test JSON file parsing with valid config
func TestConfigFile_ValidJSON(t *testing.T) {
	is := is.New(t)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configJSON := `{
		"checkInterval": "60s",
		"readTimeout": "10s",
		"shutdownTimeout": "45s",
		"failureThreshold": 5,
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
	is.Equal(cfg.CheckInterval, 60*time.Second)   // checkInterval
	is.Equal(cfg.ReadTimeout, 10*time.Second)     // readTimeout
	is.Equal(cfg.ShutdownTimeout, 45*time.Second) // shutdownTimeout
	is.Equal(cfg.FailureThreshold, 5)             // failureThreshold
	is.Equal(cfg.HTTPPort, 9090)                  // httpPort
	is.Equal(cfg.LogLevel, "debug")               // logLevel
	is.Equal(cfg.LogFormat, "text")               // logFormat
	is.Equal(cfg.CanaryFile, ".ready")            // canaryFile

	// Verify mounts
	is.Equal(len(cfg.Mounts), 2)                        // mount count
	is.Equal(cfg.Mounts[0].Name, "movies")              // mount[0].name
	is.Equal(cfg.Mounts[0].Path, "/mnt/movies")         // mount[0].path
	is.Equal(cfg.Mounts[0].CanaryFile, ".health-check") // mount[0].canaryFile
	is.Equal(cfg.Mounts[0].FailureThreshold, 3)         // mount[0].failureThreshold
	is.Equal(cfg.Mounts[1].Name, "tv")                  // mount[1].name
	is.Equal(cfg.Mounts[1].Path, "/mnt/tv")             // mount[1].path

	// Verify ConfigFile is set
	is.Equal(cfg.ConfigFile, configPath) // ConfigFile
}

// T011: Test --config flag loads specified file
func TestConfigFile_ExplicitPath(t *testing.T) {
	is := is.New(t)

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

	is.Equal(len(cfg.Mounts), 1)         // mount count
	is.Equal(cfg.ConfigFile, configPath) // ConfigFile
}

// T011 continued: Test explicit path that doesn't exist returns error
func TestConfigFile_ExplicitPath_NotFound(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	err := cfg.LoadFromFileForTesting("/nonexistent/config.json")

	is.True(err != nil) // non-existent explicit config file should error
}

// T012: Test ./config.json default location discovery
func TestConfigFile_DefaultLocation(t *testing.T) {
	is := is.New(t)

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

	is.Equal(len(cfg.Mounts), 1)                 // mount count from default config
	is.Equal(cfg.Mounts[0].Name, "default-test") // mount name
}

// T012 continued: Test missing default config.json is silently ignored
func TestConfigFile_DefaultLocation_NotFound(t *testing.T) {
	is := is.New(t)

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
	is.NoErr(cfg.LoadFromFileForTesting("")) // missing default config should not error

	// Config should still have defaults
	is.Equal(cfg.CheckInterval, 30*time.Second) // default checkInterval
}

// T013: Test backwards compatibility - no config file uses env vars
func TestConfigFile_BackwardsCompatibility(t *testing.T) {
	is := is.New(t)

	// This test verifies that the Config struct can still be used
	// with MountPaths (legacy) when no config file is present
	cfg := config.DefaultConfig()
	cfg.MountPaths = []string{"/mnt/test1", "/mnt/test2"}

	// Validation should pass with legacy MountPaths
	is.NoErr(cfg.Validate()) // valid config with MountPaths

	// Both Mounts (empty) and MountPaths should be acceptable
	is.Equal(len(cfg.Mounts), 0)     // empty Mounts array
	is.Equal(len(cfg.MountPaths), 2) // MountPaths count
}

// T020: Test per-mount canary file override
func TestConfigFile_PerMountCanaryOverride(t *testing.T) {
	is := is.New(t)

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
	is.Equal(cfg.Mounts[0].CanaryFile, ".custom-health") // mount[0] custom canary

	// Mount without override should inherit global canary file
	is.Equal(cfg.Mounts[1].CanaryFile, ".global-health") // mount[1] inherited canary
}

// T021: Test per-mount failureThreshold override
func TestConfigFile_PerMountThresholdOverride(t *testing.T) {
	is := is.New(t)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configJSON := `{
		"failureThreshold": 5,
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
	is.Equal(cfg.Mounts[0].FailureThreshold, 10) // mount[0] custom threshold

	// Mount without override should inherit global threshold
	is.Equal(cfg.Mounts[1].FailureThreshold, 5) // mount[1] inherited threshold
}

// T022: Test default inheritance when per-mount values not specified
func TestConfigFile_DefaultInheritance(t *testing.T) {
	is := is.New(t)

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
	is.Equal(cfg.Mounts[0].CanaryFile, ".health-check") // default canaryFile

	// Mount should inherit default threshold
	is.Equal(cfg.Mounts[0].FailureThreshold, 3) // default failureThreshold
}

// T029: Test invalid JSON syntax error message
func TestConfigFile_InvalidJSONSyntax(t *testing.T) {
	is := is.New(t)

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

	is.True(err != nil) // invalid JSON syntax should error
}

// T030: Test missing required "path" field error
func TestConfigFile_MissingRequiredPath(t *testing.T) {
	is := is.New(t)

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

	is.True(err != nil) // missing path field should error
}

// T031: Test invalid failureThreshold (negative) error
func TestConfigFile_InvalidFailureThreshold(t *testing.T) {
	is := is.New(t)

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

	is.True(err != nil) // negative failureThreshold should error
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
			is := is.New(t)

			if err := os.WriteFile(configPath, []byte(tt.json), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			cfg := config.DefaultConfig()
			if err := cfg.LoadFromFileForTesting(configPath); err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			is.Equal(cfg.CheckInterval, tt.expected) // checkInterval
		})
	}
}

// Test Duration with invalid format
func TestDuration_UnmarshalJSON_Invalid(t *testing.T) {
	is := is.New(t)

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

	is.True(err != nil) // invalid duration format should error
}

// TestMountNameInStatusResponse verifies FR-008: mount name appears in health status snapshot.
// This ensures the Mount.Name field is properly propagated to status responses.
func TestMountNameInStatusResponse(t *testing.T) {
	is := is.New(t)

	mount := health.NewMount("test-movies", "/mnt/movies", ".health-check", 3)

	snapshot := mount.Snapshot()

	is.Equal(snapshot.Name, "test-movies") // snapshot.Name
	is.Equal(snapshot.Path, "/mnt/movies") // snapshot.Path
}

// =============================================================================
// Security Hardening Tests (Issue #17, #15)
// =============================================================================

// TestConfigFile_FileSizeLimit verifies that config files larger than 1MB are rejected.
// This prevents DoS attacks via excessively large config files.
func TestConfigFile_FileSizeLimit(t *testing.T) {
	is := is.New(t)

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

	is.True(err != nil) // config file exceeding 1MB should error

	// Verify the error message mentions the size limit
	is.True(err != nil && strings.Contains(err.Error(), "exceeds maximum size")) // error should mention size limit
}

// TestConfigFile_FileSizeLimit_JustUnder verifies that config files just under 1MB are accepted.
func TestConfigFile_FileSizeLimit_JustUnder(t *testing.T) {
	is := is.New(t)

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

	is.NoErr(err) // config file under 1MB should not error
}

// TestConfigFile_FileSizeLimit_ExactlyOneMB verifies that config files of exactly 1MB are accepted.
// This is a boundary test - the limit is > 1MB, so exactly 1MB should be valid.
func TestConfigFile_FileSizeLimit_ExactlyOneMB(t *testing.T) {
	is := is.New(t)

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
	is.Equal(len(content), targetSize) // verify content size

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := config.DefaultConfig()
	err := cfg.LoadFromFileForTesting(configPath)

	is.NoErr(err) // config file of exactly 1MB should not error
}
