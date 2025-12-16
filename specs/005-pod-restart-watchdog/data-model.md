# Data Model: Pod Restart Watchdog

**Feature**: 005-pod-restart-watchdog
**Date**: 2025-12-15

## Overview

This document defines the data entities, state transitions, and relationships for the watchdog feature. The watchdog extends the existing health monitoring system with pod-level restart capabilities.

---

## Entities

### 1. WatchdogConfig

Configuration for watchdog behavior, extending the existing `Config` struct.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `WatchdogEnabled` | `bool` | `false` | Enable/disable watchdog mode |
| `RestartDelay` | `time.Duration` | `0s` | Delay after UNHEALTHY before triggering restart |
| `MaxRetries` | `int` | `3` | Max API retry attempts before fallback |
| `RetryBackoffInitial` | `time.Duration` | `100ms` | Initial retry delay |
| `RetryBackoffMax` | `time.Duration` | `10s` | Maximum retry delay |

**Validation Rules**:
- `RestartDelay` must be >= 0
- `MaxRetries` must be >= 1
- `RetryBackoffInitial` must be > 0
- `RetryBackoffMax` must be >= `RetryBackoffInitial`

**JSON Configuration**:
```json
{
  "watchdog": {
    "enabled": false,
    "restartDelay": "0s",
    "maxRetries": 3,
    "retryBackoffInitial": "100ms",
    "retryBackoffMax": "10s"
  }
}
```

**Environment Variables**:
- `WATCHDOG_ENABLED` - "true" or "false"
- `WATCHDOG_RESTART_DELAY` - duration string (e.g., "30s")

---

### 2. WatchdogState

Represents the current state of the watchdog state machine.

| Field | Type | Description |
|-------|------|-------------|
| `State` | `WatchdogStatus` | Current state (enum) |
| `UnhealthySince` | `*time.Time` | When mount first became unhealthy (nil if healthy) |
| `PendingMount` | `string` | Mount path that triggered pending restart |
| `RetryCount` | `int` | Current retry attempt count |
| `LastError` | `error` | Last error encountered (for logging) |

**State Enum** (`WatchdogStatus`):
```go
const (
    WatchdogDisabled      WatchdogStatus = iota // Watchdog mode off
    WatchdogArmed                               // Watching for unhealthy mounts
    WatchdogPendingRestart                      // Waiting for restart delay
    WatchdogTriggered                           // Pod deletion in progress
)
```

---

### 3. RestartEvent

Represents a watchdog-triggered restart for logging and Kubernetes events.

| Field | Type | Description |
|-------|------|-------------|
| `Timestamp` | `time.Time` | When the restart was triggered |
| `PodName` | `string` | Name of the pod being restarted |
| `Namespace` | `string` | Kubernetes namespace |
| `MountPath` | `string` | Mount that triggered the restart |
| `Reason` | `string` | Human-readable reason |
| `FailureCount` | `int` | Consecutive failures before trigger |
| `UnhealthyDuration` | `time.Duration` | How long mount was unhealthy |

---

### 4. K8sClient

Abstraction for Kubernetes API interactions.

| Field | Type | Description |
|-------|------|-------------|
| `httpClient` | `*http.Client` | HTTP client with TLS config |
| `apiServerURL` | `string` | Kubernetes API server URL |
| `token` | `string` | Bearer token for auth |
| `namespace` | `string` | Current pod namespace |
| `logger` | `*slog.Logger` | Structured logger |

**Methods**:
| Method | Returns | Description |
|--------|---------|-------------|
| `New(logger)` | `(*K8sClient, error)` | Create client with in-cluster config |
| `IsInCluster()` | `bool` | Check if running in Kubernetes |
| `CanDeletePods(ctx)` | `bool` | Validate RBAC permissions |
| `DeletePod(ctx, name)` | `error` | Delete pod via API |
| `CreateEvent(ctx, event)` | `error` | Create Kubernetes event |
| `IsPodTerminating(ctx, name)` | `bool` | Check if pod already terminating |

---

## State Transitions

### Watchdog State Machine

```
                    ┌─────────────────────────────────────────┐
                    │                                         │
                    ▼                                         │
    ┌───────────────────────────────┐                        │
    │       WatchdogDisabled        │                        │
    │   (config.WatchdogEnabled     │                        │
    │           = false)            │                        │
    └───────────────────────────────┘                        │
                    │                                         │
                    │ config.WatchdogEnabled = true           │
                    │ AND IsInCluster() = true                │
                    │ AND CanDeletePods() = true              │
                    ▼                                         │
    ┌───────────────────────────────┐                        │
    │        WatchdogArmed          │◄────────────────────────┤
    │  (monitoring for unhealthy)   │     mount recovered     │
    └───────────────────────────────┘     before delay        │
                    │                                         │
                    │ mount.Status == StatusUnhealthy         │
                    ▼                                         │
    ┌───────────────────────────────┐                        │
    │    WatchdogPendingRestart     │────────────────────────►│
    │  (waiting for RestartDelay)   │                         │
    └───────────────────────────────┘                         │
                    │                                         │
                    │ RestartDelay elapsed                    │
                    │ AND mount still unhealthy               │
                    ▼                                         │
    ┌───────────────────────────────┐                        │
    │      WatchdogTriggered        │                        │
    │   (pod deletion in progress)  │                        │
    └───────────────────────────────┘                        │
                    │                                         │
                    ├── DeletePod() succeeds ──► [Pod terminates]
                    │
                    └── DeletePod() fails after retries ──► os.Exit(1)
```

### Transition Triggers

| From | To | Trigger | Action |
|------|----|---------|--------|
| Disabled | Armed | Startup with valid config + RBAC | Log "watchdog armed" |
| Armed | PendingRestart | Mount becomes UNHEALTHY | Record `UnhealthySince`, log "restart pending" |
| PendingRestart | Armed | Mount recovers to HEALTHY | Clear `UnhealthySince`, log "restart cancelled" |
| PendingRestart | Triggered | `RestartDelay` elapsed | Create K8s event, attempt pod deletion |
| Triggered | (exit) | DeletePod succeeds | Pod terminates, controller recreates |
| Triggered | (exit) | DeletePod fails after retries | `os.Exit(1)`, liveness probe triggers restart |

---

## Relationships

```
┌─────────────┐       monitors        ┌─────────────┐
│   Monitor   │──────────────────────►│    Mount    │
└─────────────┘                       └─────────────┘
       │                                     │
       │ owns                                │ status change
       ▼                                     ▼
┌─────────────┐       observes        ┌─────────────┐
│  Watchdog   │◄──────────────────────│HealthStatus │
└─────────────┘                       └─────────────┘
       │
       │ uses
       ▼
┌─────────────┐       calls           ┌─────────────┐
│  K8sClient  │──────────────────────►│ K8s API     │
└─────────────┘                       └─────────────┘
```

### Integration Points

1. **Monitor → Watchdog**: Monitor notifies watchdog on state transitions
2. **Watchdog → K8sClient**: Watchdog uses client for pod deletion and events
3. **Config → Watchdog**: Watchdog reads configuration at startup
4. **Main → Watchdog**: Main initializes watchdog with dependencies

---

## Logging Events

All watchdog state changes produce structured log entries:

| Event | Level | Fields |
|-------|-------|--------|
| Watchdog armed | INFO | `event=watchdog_armed`, `pod`, `namespace` |
| Watchdog disabled (no RBAC) | WARN | `event=watchdog_disabled`, `reason=rbac_missing` |
| Watchdog disabled (not in cluster) | INFO | `event=watchdog_disabled`, `reason=not_in_cluster` |
| Restart pending | WARN | `event=restart_pending`, `mount_path`, `delay` |
| Restart cancelled | INFO | `event=restart_cancelled`, `mount_path`, `reason=mount_recovered` |
| Restart triggered | WARN | `event=restart_triggered`, `mount_path`, `unhealthy_duration` |
| Pod deletion success | INFO | `event=pod_deleted`, `pod`, `namespace` |
| Pod deletion retry | WARN | `event=pod_deletion_retry`, `attempt`, `error` |
| Pod deletion failed | ERROR | `event=pod_deletion_failed`, `retries`, `error` |
| Fallback exit | ERROR | `event=fallback_exit`, `reason=api_failure` |
