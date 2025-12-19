// Package testutil provides shared test utilities for the mount-monitor project.
package testutil

import (
	"io"
	"log/slog"
	"testing"
)

// Logger returns a silent slog.Logger for tests.
// The logger discards all output to avoid cluttering test output.
// This is useful when testing components that require a logger but
// where the log output is not relevant to the test assertions.
func Logger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
