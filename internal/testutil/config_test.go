package testutil_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/cscheib/debrid-mount-monitor/internal/config"
	"github.com/cscheib/debrid-mount-monitor/internal/testutil"
	"github.com/matryer/is"
)

func TestTempConfig(t *testing.T) {
	is := is.New(t)

	cfg := &config.Config{
		HTTPPort: 8080,
		Mounts: []config.MountConfig{{
			Name:       "test-mount",
			Path:       "/tmp/test",
			CanaryFile: ".health-check",
		}},
	}

	path := testutil.TempConfig(t, cfg)

	// File should exist
	_, err := os.Stat(path)
	is.NoErr(err) // Config file should exist

	// File should contain valid JSON that matches the config
	data, err := os.ReadFile(path)
	is.NoErr(err) // Should be able to read config file

	var loaded config.Config
	err = json.Unmarshal(data, &loaded)
	is.NoErr(err) // Config should be valid JSON

	is.Equal(loaded.HTTPPort, cfg.HTTPPort)       // HTTPPort should match
	is.Equal(len(loaded.Mounts), len(cfg.Mounts)) // Mount count should match
	is.Equal(loaded.Mounts[0].Name, cfg.Mounts[0].Name)
}

func TestTempConfig_MultipleConfigs(t *testing.T) {
	is := is.New(t)

	cfg1 := &config.Config{HTTPPort: 8081}
	cfg2 := &config.Config{HTTPPort: 8082}

	path1 := testutil.TempConfig(t, cfg1)
	path2 := testutil.TempConfig(t, cfg2)

	// Paths should be different
	is.True(path1 != path2) // Each config should have unique path
}
