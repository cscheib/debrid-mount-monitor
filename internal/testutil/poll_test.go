package testutil_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/cscheib/debrid-mount-monitor/internal/testutil"
	"github.com/matryer/is"
)

func TestPollUntil_ImmediateSuccess(t *testing.T) {
	is := is.New(t)

	var callCount int32
	start := time.Now()

	testutil.PollUntil(t, time.Second, func() bool {
		atomic.AddInt32(&callCount, 1)
		return true // Immediate success
	})

	elapsed := time.Since(start)
	is.True(elapsed < 100*time.Millisecond) // Should complete quickly
	is.Equal(atomic.LoadInt32(&callCount), int32(1))
}

func TestPollUntil_EventualSuccess(t *testing.T) {
	is := is.New(t)

	var callCount int32
	targetCalls := int32(5)

	testutil.PollUntil(t, time.Second, func() bool {
		count := atomic.AddInt32(&callCount, 1)
		return count >= targetCalls
	})

	is.True(atomic.LoadInt32(&callCount) >= targetCalls) // Should have called multiple times
}

func TestPollUntil_ExternalCondition(t *testing.T) {
	is := is.New(t)

	var ready atomic.Bool

	// Simulate async operation completing
	go func() {
		time.Sleep(50 * time.Millisecond)
		ready.Store(true)
	}()

	start := time.Now()
	testutil.PollUntil(t, time.Second, func() bool {
		return ready.Load()
	})
	elapsed := time.Since(start)

	is.True(ready.Load())                   // Condition should be met
	is.True(elapsed >= 50*time.Millisecond) // Should wait for condition
	is.True(elapsed < 500*time.Millisecond) // Should not take too long
}
