package testutil

import (
	"testing"
	"time"
)

// PollUntil repeatedly calls the condition function until it returns true
// or the timeout expires. If the timeout is reached, the test fails.
//
// The polling interval starts at 10ms and increases exponentially up to 100ms.
// This balances responsiveness for fast operations with efficiency for slower ones.
//
// Example:
//
//	testutil.PollUntil(t, 5*time.Second, func() bool {
//	    return service.Status() == "ready"
//	})
func PollUntil(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	interval := 10 * time.Millisecond
	maxInterval := 100 * time.Millisecond

	for {
		if condition() {
			return
		}

		// Check remaining time before sleeping
		remaining := time.Until(deadline)
		if remaining <= 0 {
			t.Fatalf("condition not met within %v timeout", timeout)
		}

		// Don't sleep longer than remaining time
		sleepDuration := interval
		if sleepDuration > remaining {
			sleepDuration = remaining
		}
		time.Sleep(sleepDuration)

		// Exponential backoff up to maxInterval
		interval = interval * 2
		if interval > maxInterval {
			interval = maxInterval
		}
	}
}
