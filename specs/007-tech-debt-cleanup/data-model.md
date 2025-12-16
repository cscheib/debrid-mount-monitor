# Data Model: Tech Debt Cleanup

**Feature**: 007-tech-debt-cleanup
**Date**: 2025-12-16

## Overview

This feature modifies existing data structures (configuration) rather than introducing new entities. This document captures the before/after state for terminology alignment.

## Configuration Structure Changes

### Global Config (internal/config/config.go)

**Before**:
```go
type Config struct {
    // ... other fields ...
    DebounceThreshold int  // Default consecutive failures before unhealthy
}
```

**After**:
```go
type Config struct {
    // ... other fields ...
    FailureThreshold int  // Default consecutive failures before unhealthy
}
```

### Mount Config (internal/config/config.go)

**Unchanged** (already uses correct terminology):
```go
type MountConfig struct {
    Path             string  // Filesystem path to mount point (required)
    CanaryFile       string  // Relative path to canary file (optional)
    Name             string  // Human-readable mount name (optional)
    FailureThreshold int     // Per-mount override (0 = use global)
}
```

### JSON File Config (internal/config/file.go)

**Before**:
```go
type fileConfig struct {
    // ... other fields ...
    DebounceThreshold int `json:"debounceThreshold,omitempty"`
}
```

**After**:
```go
type fileConfig struct {
    // ... other fields ...
    FailureThreshold int `json:"failureThreshold,omitempty"`
}
```

### JSON Schema Change

**Before** (config.json):
```json
{
  "debounceThreshold": 3,
  "mounts": [...]
}
```

**After** (config.json):
```json
{
  "failureThreshold": 3,
  "mounts": [...]
}
```

## Mount State (internal/health/state.go)

**Unchanged** structure, but comments updated:

```go
type Mount struct {
    // ... fields ...
    FailureCount int  // Consecutive failure count (compared against failureThreshold)
}
```

Comment changes only - terminology alignment in descriptions.

## CLI Flag Changes

| Before | After | Default |
|--------|-------|---------|
| `--debounce-threshold` | `--failure-threshold` | 3 |

## Environment Variables (Removed)

The following environment variables are removed from configuration processing:

| Variable | Replacement |
|----------|-------------|
| `MOUNT_PATHS` | `mounts[].path` in JSON config or `--mount-paths` flag |
| `CANARY_FILE` | `canaryFile` in JSON config or `--canary-file` flag |
| `CHECK_INTERVAL` | `checkInterval` in JSON config or `--check-interval` flag |
| `READ_TIMEOUT` | `readTimeout` in JSON config or `--read-timeout` flag |
| `SHUTDOWN_TIMEOUT` | `shutdownTimeout` in JSON config or `--shutdown-timeout` flag |
| `DEBOUNCE_THRESHOLD` | `failureThreshold` in JSON config or `--failure-threshold` flag |
| `HTTP_PORT` | `httpPort` in JSON config or `--http-port` flag |
| `LOG_LEVEL` | `logLevel` in JSON config or `--log-level` flag |
| `LOG_FORMAT` | `logFormat` in JSON config or `--log-format` flag |
| `WATCHDOG_ENABLED` | `watchdog.enabled` in JSON config |
| `WATCHDOG_RESTART_DELAY` | `watchdog.restartDelay` in JSON config |

## Validation Rules

No changes to validation logic, only field name changes:

- `failureThreshold` must be >= 1 (error message updated from "debounce threshold must be >= 1")

## Migration Notes

Users must update their `config.json` files:

```diff
{
- "debounceThreshold": 3,
+ "failureThreshold": 3,
  "mounts": [...]
}
```

Users using environment variables must migrate to JSON config or CLI flags.
