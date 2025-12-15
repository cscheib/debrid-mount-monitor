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

	// Perform the canary file read in a goroutine to respect context cancellation.
	//
	// NOTE: os.ReadFile does not respect context cancellation. If the context times out
	// before ReadFile completes (e.g., on a hung NFS mount), this goroutine will leak
	// until the underlying I/O operation eventually completes or fails. This is a known
	// limitation of Go's file I/O - there is no portable way to cancel a blocking read.
	// In practice, this is acceptable because:
	// 1. Leaked goroutines will eventually complete when the mount recovers
	// 2. The memory overhead per goroutine is small (~2KB stack)
	// 3. Alternative approaches (goroutine pools) add complexity without solving the root cause
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
