package health

import (
	"context"
	"os"
	"time"
)

// Checker performs health checks on mount points.
type Checker struct {
	timeout time.Duration
}

// NewChecker creates a new Checker with the specified read timeout.
func NewChecker(timeout time.Duration) *Checker {
	return &Checker{
		timeout: timeout,
	}
}

// Check performs a health check on the given mount by reading its canary file.
// It returns a CheckResult indicating success or failure.
func (c *Checker) Check(ctx context.Context, mount *Mount) *CheckResult {
	start := time.Now()

	result := &CheckResult{
		Mount:     mount,
		Timestamp: start,
	}

	// Create a timeout context for the check
	checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Perform the canary file read in a goroutine to respect context cancellation
	done := make(chan error, 1)
	go func() {
		_, err := os.ReadFile(mount.CanaryPath)
		done <- err
	}()

	// Wait for either completion or context cancellation
	select {
	case err := <-done:
		result.Duration = time.Since(start)
		if err != nil {
			result.Success = false
			result.Error = err
		} else {
			result.Success = true
		}
	case <-checkCtx.Done():
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = checkCtx.Err()
	}

	return result
}
