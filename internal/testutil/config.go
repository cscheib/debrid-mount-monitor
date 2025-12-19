package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cscheib/debrid-mount-monitor/internal/config"
)

// TempConfig creates a temporary config file with the given configuration
// and returns the path to the file. The file is automatically cleaned up
// when the test completes.
//
// Example:
//
//	cfg := &config.Config{
//	    HTTPPort: 8080,
//	    Mounts: []config.MountConfig{{
//	        Name:       "test",
//	        Path:       t.TempDir(),
//	        CanaryFile: ".health-check",
//	    }},
//	}
//	configPath := testutil.TempConfig(t, cfg)
func TempConfig(t *testing.T, cfg *config.Config) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	return path
}
