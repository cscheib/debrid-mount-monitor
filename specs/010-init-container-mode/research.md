# Research: Init Container Mode

**Feature**: 010-init-container-mode
**Date**: 2025-12-17

## Summary

This feature requires no external research. All implementation details leverage existing codebase patterns and approved dependencies.

## Decision Log

### 1. Check Execution Strategy

**Decision**: Sequential mount checks

**Rationale**:
- Matches existing `monitor.Monitor` behavior which iterates mounts sequentially
- Simpler implementation with predictable timing
- Total worst-case time = N × ReadTimeout (acceptable for typical 1-5 mounts with 5s timeout)
- Parallel execution would add complexity without significant benefit for this use case

**Alternatives Considered**:
- Parallel checks via goroutines: Rejected (unnecessary complexity, harder to reason about timeout behavior)

### 2. Exit Code Strategy

**Decision**: Binary exit codes (0 = success, 1 = failure)

**Rationale**:
- Standard Unix convention
- Kubernetes initContainer only needs success/failure distinction
- No practical benefit to differentiated error codes (2 for config error, 3 for timeout, etc.)

**Alternatives Considered**:
- Differentiated exit codes: Rejected (adds complexity without benefit; logs provide sufficient detail)

### 3. Configuration Validation

**Decision**: Skip irrelevant validations in init mode

**Rationale**:
- HTTPPort, CheckInterval, ShutdownTimeout, Watchdog config are not used in init mode
- Requiring valid values for unused settings would be confusing
- Core validations (mounts, ReadTimeout, LogLevel) still apply

**Implementation**:
- Add `InitContainerMode bool` to Config struct
- Pass to `Validate()` or create `ValidateForInitMode()` variant

### 4. Logging Approach

**Decision**: Reuse existing slog-based structured logging

**Rationale**:
- Consistent with rest of application
- Already configured for stdout/stderr routing
- Supports both JSON and text formats

**No Changes Required**: setupLogger() function works as-is

## Existing Components to Reuse

| Component | Location | Purpose in Init Mode |
|-----------|----------|---------------------|
| `health.Checker` | `internal/health/checker.go` | Perform canary file checks |
| `health.Mount` | `internal/health/state.go` | Mount representation |
| `config.Load()` | `internal/config/config.go` | Configuration loading |
| `setupLogger()` | `cmd/mount-monitor/main.go` | Logger setup |

## No External Research Required

- **pflag**: Already approved and in use for flag parsing
- **slog**: Standard library (Go 1.21+), no external dependency
- **os.Exit()**: Standard library, standard Unix exit code semantics
- **context.Background()**: Standard library for check timeout context
