package testutil_test

import (
	"testing"

	"github.com/cscheib/debrid-mount-monitor/internal/testutil"
	"github.com/matryer/is"
)

func TestLogger(t *testing.T) {
	is := is.New(t)

	logger := testutil.Logger(t)

	is.True(logger != nil) // Logger should return non-nil logger

	// Logger should not panic when used
	logger.Info("test message", "key", "value")
	logger.Error("error message", "err", "test error")
}
