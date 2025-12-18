# Quickstart: Testing Quality Improvements

**Feature**: 011-testing-improvements
**Date**: 2025-12-18

## Overview

This guide helps developers use the enhanced testing infrastructure including goroutine leak detection, race condition testing, and coverage reporting.

## Prerequisites

- Go 1.21 or later
- Docker (for KIND E2E tests)
- KIND (for E2E tests): `go install sigs.k8s.io/kind@latest`

## Quick Commands

| Task | Command |
|------|---------|
| Run all tests | `make test` |
| Run tests with race detection | `make test-race` |
| Generate coverage report | `make test-cover` |
| Run tests + race + coverage | `make test-all` |
| Run E2E tests in KIND | `make kind-test` |

## Using goleak for Goroutine Leak Detection

### Per-Test Verification (Recommended)

Add `defer goleak.VerifyNone(t)` at the start of tests that spawn goroutines:

```go
import "go.uber.org/goleak"

func TestMonitorLoop(t *testing.T) {
    defer goleak.VerifyNone(t)

    // Test code that spawns goroutines
    monitor := NewMonitor(cfg)
    ctx, cancel := context.WithCancel(context.Background())
    go monitor.Start(ctx)

    // ... test assertions ...

    cancel() // Clean up goroutines before test ends
}
```

### Ignoring Known Goroutines

Some goroutines (e.g., from testing infrastructure) are expected. Ignore them:

```go
func TestWithExpectedGoroutines(t *testing.T) {
    defer goleak.VerifyNone(t,
        goleak.IgnoreTopFunction("net/http.(*Server).Serve"),
    )
    // ...
}
```

## Using testutil Helpers

### Logger

Get a silent logger for tests (no output noise):

```go
import "github.com/cscheib/debrid-mount-monitor/internal/testutil"

func TestWithLogging(t *testing.T) {
    logger := testutil.Logger(t)
    service := NewService(logger)
    // ...
}
```

### TempConfig

Create a temporary config file for testing:

```go
func TestWithConfig(t *testing.T) {
    cfg := &config.Config{
        HTTPPort: 8080,
        Mounts: []config.MountConfig{{
            Name:       "test",
            Path:       t.TempDir(),
            CanaryFile: ".health-check",
        }},
    }
    configPath := testutil.TempConfig(t, cfg)
    // configPath is automatically cleaned up after test
}
```

### PollUntil

Wait for async conditions with timeout:

```go
func TestAsyncOperation(t *testing.T) {
    service.Start()

    testutil.PollUntil(t, 5*time.Second, func() bool {
        return service.Status() == "ready"
    })

    // Test continues after condition is met
}
```

## Running Race Detection

Race detection is enabled by default in CI. For local development:

```bash
# Quick race check
make test-race

# Or manually
go test -race ./...
```

### Handling Race Detection Failures

If a race is detected:

1. The test output will show the race location
2. Look for concurrent access to shared state
3. Add proper synchronization (mutex, channel, atomic)
4. Re-run with `-race` to verify fix

## Coverage Reporting

### Generate HTML Report

```bash
make test-cover
open coverage.html
```

### Check Package Coverage

```bash
go test -coverprofile=c.out ./internal/watchdog/...
go tool cover -func=c.out
```

### Coverage Targets

| Package | Target |
|---------|--------|
| Overall | ≥ 75% |
| internal/watchdog | ≥ 80% |
| cmd/mount-monitor | ≥ 60% |

## E2E Testing with KIND

### Quick E2E Test

```bash
# Runs test, creates/deletes cluster automatically
make kind-test
```

### Manual E2E Workflow

```bash
# Create cluster
make kind-create

# Build and load image
make kind-load

# Deploy
make kind-deploy

# Watch logs
make kind-logs

# Simulate mount failure
POD=$(kubectl -n mount-monitor-dev get pod -l app=test-app-with-monitor -o name)
kubectl -n mount-monitor-dev exec $POD -c main-app -- rm /mnt/test/.health-check

# Observe watchdog behavior in logs

# Restore mount
kubectl -n mount-monitor-dev exec $POD -c main-app -- sh -c 'echo healthy > /mnt/test/.health-check'

# Clean up
make kind-delete
```

## Writing New Tests

### Test Structure

Follow the existing pattern:

```go
func TestSomething(t *testing.T) {
    // 1. Arrange - set up test fixtures
    logger := testutil.Logger(t)
    cfg := &config.Config{...}

    // 2. Act - perform the action being tested
    result, err := DoSomething(cfg, logger)

    // 3. Assert - verify the outcome
    is := is.New(t)
    is.NoErr(err)
    is.Equal(result, expected)
}
```

### Table-Driven Tests

For multiple scenarios:

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid", "good-input", false},
        {"invalid", "bad-input", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := Validate(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Troubleshooting

### goleak Reports Unexpected Goroutines

1. Check if goroutines are properly cleaned up (context cancelled, channels closed)
2. Use `goleak.IgnoreTopFunction()` for known background goroutines
3. Add cleanup in `t.Cleanup()` or `defer`

### Race Detector False Positives

Rare, but if you believe it's a false positive:
1. Verify the code is actually safe
2. Consider if the synchronization is clear to the race detector
3. Document and skip with `-race` only as last resort

### Coverage Seems Low

1. Check if tests are in `_test.go` files
2. Verify test functions start with `Test`
3. Check if code paths have tests (use HTML report)
4. Look for untested error handling paths

## Next Steps

- Run `make test-all` to verify everything passes
- Add goleak to tests that spawn goroutines
- Use testutil helpers in new tests
- Aim for coverage targets in new code
