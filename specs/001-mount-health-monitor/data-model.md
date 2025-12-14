# Data Model: Mount Health Monitor

**Date**: 2025-12-14
**Feature**: 001-mount-health-monitor

## Overview

This service is stateless by design. All data structures exist only in memory and are lost on restart. This is intentional - Kubernetes manages the lifecycle, and there's no need to persist health state across restarts.

## Entities

### 1. Config

Runtime configuration parsed from environment variables and command-line flags.

```go
type Config struct {
    // Mount configuration
    MountPaths       []string      // Paths to monitor
    CanaryFile       string        // Relative path to canary file within each mount

    // Timing configuration
    CheckInterval    time.Duration // Time between health checks
    ReadTimeout      time.Duration // Timeout for canary file read
    ShutdownTimeout  time.Duration // Max time for graceful shutdown

    // Debounce configuration
    DebounceThreshold int          // Consecutive failures before unhealthy

    // Server configuration
    HTTPPort         int           // Port for health endpoints

    // Logging configuration
    LogLevel         string        // debug, info, warn, error
    LogFormat        string        // json, text
}
```

**Validation Rules**:
- `MountPaths` must have at least one entry
- `CheckInterval` must be >= 1 second
- `ReadTimeout` must be >= 100 milliseconds and < `CheckInterval`
- `DebounceThreshold` must be >= 1
- `HTTPPort` must be 1-65535
- `LogLevel` must be one of: debug, info, warn, error
- `LogFormat` must be one of: json, text

### 2. Mount

Represents a single mount point being monitored.

```go
type Mount struct {
    Path           string        // Absolute path to mount point
    CanaryPath     string        // Full path to canary file (Path + CanaryFile)
    Status         HealthStatus  // Current health status
    LastCheck      time.Time     // Timestamp of last health check
    LastError      error         // Last error encountered (nil if healthy)
    FailureCount   int           // Consecutive failure count (for debounce)
}
```

**State Transitions**:
```
                    ┌─────────────┐
                    │   UNKNOWN   │  (initial state)
                    └──────┬──────┘
                           │ first check
                           ▼
          ┌────────────────┴────────────────┐
          │                                 │
          ▼                                 ▼
    ┌───────────┐                    ┌─────────────┐
    │  HEALTHY  │◄───────────────────│  DEGRADED   │
    │           │    check passes    │ (count < N) │
    └─────┬─────┘                    └──────┬──────┘
          │                                 │
          │ check fails                     │ check fails
          │                                 │ count >= threshold
          ▼                                 ▼
    ┌───────────┐                    ┌─────────────┐
    │  DEGRADED │                    │  UNHEALTHY  │
    │ (count=1) │                    │             │
    └───────────┘                    └──────┬──────┘
                                           │
                                           │ check passes
                                           ▼
                                     ┌───────────┐
                                     │  HEALTHY  │
                                     └───────────┘
```

### 3. HealthStatus

Enumeration of possible health states.

```go
type HealthStatus int

const (
    StatusUnknown   HealthStatus = iota  // Initial state, no check performed yet
    StatusHealthy                         // Mount is accessible
    StatusDegraded                        // Mount has failed but within debounce threshold
    StatusUnhealthy                       // Mount has failed past debounce threshold
)
```

### 4. CheckResult

The outcome of a single health check operation.

```go
type CheckResult struct {
    Mount      *Mount        // Reference to the mount checked
    Timestamp  time.Time     // When the check was performed
    Success    bool          // Whether the canary file was readable
    Duration   time.Duration // How long the check took
    Error      error         // Error if check failed (nil on success)
}
```

### 5. StateTransition

Records a change in health status for logging/observability.

```go
type StateTransition struct {
    Mount         *Mount       // Reference to the mount
    Timestamp     time.Time    // When the transition occurred
    PreviousState HealthStatus // State before transition
    NewState      HealthStatus // State after transition
    Trigger       string       // What caused the transition (e.g., "check_failed", "recovered")
}
```

### 6. ProbeResponse

The response structure for Kubernetes probe endpoints.

```go
type ProbeResponse struct {
    Status    string            `json:"status"`     // "healthy" or "unhealthy"
    Timestamp string            `json:"timestamp"`  // ISO 8601 format
    Mounts    []MountStatus     `json:"mounts"`     // Per-mount status
}

type MountStatus struct {
    Path         string `json:"path"`
    Status       string `json:"status"`       // "healthy", "degraded", "unhealthy", "unknown"
    LastCheck    string `json:"last_check"`   // ISO 8601 timestamp
    FailureCount int    `json:"failure_count"`
    Error        string `json:"error,omitempty"` // Last error message if any
}
```

## Relationships

```
Config
  │
  └──> []Mount (one Config has many Mounts)

Mount
  │
  ├──> HealthStatus (current state)
  │
  └──> CheckResult (latest check, not persisted historically)

Monitor (runtime orchestrator)
  │
  ├──> Config
  ├──> []Mount
  └──> Server (HTTP endpoints)
```

## Data Flow

1. **Startup**: Config parsed → Mounts created (one per path) → Monitor starts
2. **Health Check Loop**:
   - For each Mount: perform check → update Mount state → emit StateTransition if changed
   - Sleep for CheckInterval
   - Repeat
3. **Probe Request**:
   - Liveness: Return 200 if all Mounts are HEALTHY or DEGRADED, else 503
   - Readiness: Return 200 if all Mounts are HEALTHY, else 503
4. **Shutdown**: Stop check loop → drain HTTP connections → exit

## Memory Estimates

| Entity | Size | Count | Total |
|--------|------|-------|-------|
| Config | ~500 bytes | 1 | 500 B |
| Mount | ~200 bytes | 1-10 | 2 KB |
| CheckResult | ~100 bytes | 1-10 (latest only) | 1 KB |
| HTTP buffers | ~64 KB | per connection | 64-256 KB |
| **Total** | | | **< 1 MB typical** |

With Go runtime overhead, expect 5-10 MB total memory usage.
