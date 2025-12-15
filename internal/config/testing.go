// Package config - test helpers
//
// This file exports internal functions for use in tests.
// These functions should not be used in production code.
package config

// LoadFromFileForTesting exposes loadFromFile for unit tests.
// This allows tests in other packages (e.g., tests/unit/) to test
// file loading behavior without using the full Load() function.
//
// WARNING: This function is intended for testing only.
// Do not use in production code.
func (c *Config) LoadFromFileForTesting(configPath string) error {
	return c.loadFromFile(configPath)
}
