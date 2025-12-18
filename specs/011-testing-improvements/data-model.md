# Data Model: Testing Quality Improvements

**Feature**: 011-testing-improvements
**Date**: 2025-12-18

## Overview

This feature deals with testing infrastructure rather than persistent data. The "entities" here represent test concepts and coverage metrics.

## Test Entities

### TestSuite

Represents the collection of all tests in the project.

| Attribute | Type | Description |
|-----------|------|-------------|
| packages | []Package | List of packages with tests |
| totalCoverage | float64 | Overall coverage percentage |
| raceDetected | bool | Whether race conditions were found |
| goroutineLeaks | []string | List of leaked goroutine stacks |

### Package

Represents a testable Go package.

| Attribute | Type | Description |
|-----------|------|-------------|
| path | string | Package import path (e.g., `internal/watchdog`) |
| coverage | float64 | Package-specific coverage percentage |
| testCount | int | Number of test functions |
| hasGoleak | bool | Whether goleak verification is enabled |

### CoverageReport

Represents a coverage analysis result.

| Attribute | Type | Description |
|-----------|------|-------------|
| timestamp | time.Time | When report was generated |
| overall | float64 | Total coverage percentage |
| byPackage | map[string]float64 | Coverage per package |
| byFunction | map[string]float64 | Coverage per function |
| htmlPath | string | Path to HTML report |

### TestHelper

Represents a shared test utility function.

| Attribute | Type | Description |
|-----------|------|-------------|
| name | string | Function name (e.g., `Logger`, `TempConfig`) |
| package | string | Always `internal/testutil` |
| signature | string | Function signature |
| purpose | string | What the helper does |

## Coverage Targets

| Package | Current | Target | Status |
|---------|---------|--------|--------|
| `internal/server` | 89.5% | 90%+ | ✅ Achieved |
| `internal/health` | 85.9% | 86%+ | ✅ Achieved |
| `internal/monitor` | 79.1% | 80%+ | 🔄 In progress |
| `internal/config` | 75.7% | 76%+ | ✅ Achieved |
| `internal/watchdog` | 45.2% | 80%+ | ❌ Critical gap |
| `cmd/mount-monitor` | 0.0% | 60%+ | ❌ Critical gap |
| **Overall** | 57.3% | 75%+ | 🔄 In progress |

## testutil Package API

### Logger

```go
// Logger returns a silent slog.Logger for testing
func Logger(t *testing.T) *slog.Logger
```

**Relationships**: Used by all test files that need logging

### TempConfig

```go
// TempConfig creates a temporary config file and returns its path
// The file is automatically cleaned up when the test completes
func TempConfig(t *testing.T, cfg *config.Config) string
```

**Relationships**: Uses `config.Config` entity from `internal/config`

### PollUntil

```go
// PollUntil repeatedly calls condition until it returns true or timeout expires
// Fails the test if timeout is reached
func PollUntil(t *testing.T, timeout time.Duration, condition func() bool)
```

**Relationships**: Used by async tests (monitor, watchdog, server lifecycle)

## State Transitions (Watchdog Tests)

The watchdog has a state machine that needs comprehensive testing:

```
┌──────────┐      mount healthy      ┌────────┐
│ Disabled │ ─────────────────────► │ Armed  │
└──────────┘                         └────────┘
                                         │
                                         │ mount unhealthy
                                         ▼
                                   ┌──────────────┐
                                   │PendingRestart│
                                   └──────────────┘
                                         │
                    mount recovers  ◄────┤
                    (cancel restart)     │ delay expires
                         │               ▼
                         │          ┌───────────┐
                         └────────► │ Triggered │
                         (→Armed)   └───────────┘
                                         │
                                         │ pod deleted
                                         ▼
                                    ┌─────────┐
                                    │  Armed  │
                                    └─────────┘
```

**Test Coverage Required**:
- All state transitions
- Concurrent failure/recovery (race conditions)
- Timer cancellation on recovery
- API retry exhaustion

## E2E Test Scenarios

| Scenario ID | Description | Entities Involved |
|-------------|-------------|-------------------|
| E2E-001 | Mount failure detection | KIND cluster, Pod, Mount |
| E2E-002 | Recovery cancels restart | KIND cluster, Pod, Watchdog timer |
| E2E-003 | Pod restart execution | KIND cluster, Pod, K8s API |

## Validation Rules

1. **Coverage Threshold**: Overall coverage must be ≥ 75%
2. **Package Coverage**: Watchdog ≥ 80%, main.go ≥ 60%
3. **Race Detection**: Zero races when running with `-race`
4. **Goroutine Leaks**: Zero leaks detected by goleak
5. **E2E Scenarios**: All 3 scenarios must pass in KIND

## Notes

- This is testing infrastructure, not production data
- No persistent storage required
- Coverage metrics are computed at test time, not stored
- testutil helpers are compile-time only (test binary)
