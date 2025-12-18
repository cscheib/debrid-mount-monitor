# Research: Testing Quality Improvements

**Feature**: 011-testing-improvements
**Date**: 2025-12-18
**Status**: Complete

## Research Topics

### 1. goleak Integration Patterns

**Decision**: Use `goleak.VerifyNone(t)` at the end of individual tests, not `TestMain`

**Rationale**:
- Per-test verification provides clearer failure messages identifying which test leaked
- TestMain approach runs once for the entire package, making leak sources harder to identify
- Can selectively apply to tests that spawn goroutines (watchdog, monitor, server)
- Uber's recommended pattern for most use cases

**Alternatives Considered**:
| Pattern | Pros | Cons |
|---------|------|------|
| `goleak.VerifyNone(t)` per test | Clear attribution, selective use | More boilerplate |
| `goleak.VerifyTestMain(m)` | Single setup, catches all leaks | Hard to attribute, runs for all tests |
| Custom goroutine counting | No dependency | Complex, error-prone |

**Implementation Pattern**:
```go
func TestWatchdogStart(t *testing.T) {
    defer goleak.VerifyNone(t)
    // test code
}
```

---

### 2. Existing CI Race Detection

**Decision**: Race detection already enabled in CI (line 29 of ci.yml)

**Rationale**: No changes needed for race detection in CI. The current workflow uses:
```yaml
run: go test -v -race -coverprofile=coverage.out -coverpkg=./internal/...,./cmd/... ./...
```

**Action Items**:
- Add `test-race` Makefile target for local development convenience
- No CI changes required for race detection

---

### 3. Coverage Reporting Strategy

**Decision**: Keep current CI approach (50% threshold), add HTML report for local dev

**Rationale**:
- CI already enforces 50% minimum (line 31-40 of ci.yml)
- User chose "no enforcement yet" for stricter thresholds
- HTML reports help developers identify gaps locally

**Implementation**:
- Makefile `test-cover` target generates HTML report
- CI continues with existing coverage workflow
- Coverage artifact already available via `-coverprofile`

---

### 4. testutil Package Design

**Decision**: Create `internal/testutil/` with three focused helpers

**Rationale**:
- Existing tests duplicate logger creation, polling logic, and config setup
- Consolidation reduces boilerplate and ensures consistency
- `internal/` prevents external use while allowing cross-package sharing

**API Design**:

| Function | Signature | Purpose |
|----------|-----------|---------|
| `Logger()` | `func Logger(t *testing.T) *slog.Logger` | Returns silent logger for tests |
| `TempConfig()` | `func TempConfig(t *testing.T, cfg *config.Config) string` | Creates temp config file, returns path |
| `PollUntil()` | `func PollUntil(t *testing.T, timeout time.Duration, condition func() bool)` | Polls condition with timeout |

---

### 5. main.go Testability Refactoring

**Decision**: Extract `setupLogger()` and `runInitMode()` as package-level functions

**Rationale**:
- Current main() is monolithic, making testing difficult
- Extracting these functions allows unit testing without running full main()
- Init mode is self-contained and testable with temp directories
- Logger setup can be tested by capturing stdout/stderr

**Refactoring Plan**:
1. `setupLogger(level string, format string) *slog.Logger` - returns configured logger
2. `runInitMode(ctx context.Context, cfg *config.Config, logger *slog.Logger) error` - extracted init logic
3. Keep main() as thin orchestrator

---

### 6. Watchdog Test Coverage Gaps

**Decision**: Focus on state machine transitions and timer behavior

**Rationale**: Current 45% coverage misses critical paths:
- Timer cancellation when mount recovers during pending restart
- Race between mount failure and recovery notifications
- API retry exhaustion leading to process exit
- Pod-already-terminating check

**Test Scenarios to Add**:
| Scenario | Current Coverage | Priority |
|----------|-----------------|----------|
| Recovery cancels pending restart | ❌ Missing | High |
| Concurrent failure/recovery race | ❌ Missing | High |
| API retry exhaustion → exit | ❌ Missing | High |
| Pod already terminating | ❌ Missing | Medium |
| State machine all transitions | Partial | Medium |

---

### 7. E2E Test Architecture

**Decision**: Extend existing KIND infrastructure with programmatic Go tests

**Rationale**:
- `kind-test` target already exists with basic e2e-test.sh
- Adding Go-based E2E tests in `tests/e2e/` provides better assertions
- Can reuse existing manifest patterns from `deploy/kind/`
- Go tests integrate with coverage reporting

**Test Scenarios**:
1. **Mount Failure Detection**: Remove canary → verify unhealthy transition
2. **Recovery Cancellation**: Restore canary during pending restart → verify restart cancelled
3. **Pod Restart**: Allow restart to complete → verify pod deleted, event created

---

## Unresolved Items

None - all research items resolved. Ready to proceed with implementation.

## References

- [goleak documentation](https://github.com/uber-go/goleak)
- [Go testing best practices](https://go.dev/doc/tutorial/add-a-test)
- [Go race detector](https://go.dev/doc/articles/race_detector)
- [KIND documentation](https://kind.sigs.k8s.io/)
