# Data Model: Development Tooling Improvements

**Feature**: 006-dev-tooling-improvements
**Date**: 2025-12-16

## Overview

This feature is primarily infrastructure/tooling focused with minimal data model requirements. The primary "data" involved is test configuration and verification state.

---

## Test Configuration

### E2E Test Config (ConfigMap)

Used in `deploy/kind/test-configmap.yaml` for faster test execution.

| Field | Type | Test Value | Production Value | Notes |
|-------|------|------------|------------------|-------|
| `check_interval` | duration | 2s | 30s | Faster feedback loop |
| `failure_threshold` | int | 2 | 3 | Fewer checks before unhealthy |
| `watchdog.enabled` | bool | true | false | Must be enabled for test |
| `watchdog.restart_delay` | duration | 5s | 0s | Visible delay for verification |

### Environment Variables (Test Context)

| Variable | Test Value | Purpose |
|----------|------------|---------|
| `KIND_CLUSTER_NAME` | `debrid-mount-monitor-test` | Isolated test cluster |
| `KIND_NAMESPACE` | `watchdog-e2e-test` | Isolated test namespace |
| `KEEP_CLUSTER` | `0` (default) or `1` | Preserve cluster for debugging |

---

## Verification State

### Pod Restart Verification

State captured during E2E test execution:

| Field | Type | Example | Purpose |
|-------|------|---------|---------|
| `pod_name` | string | `test-app-with-monitor-abc123` | Pod being tested |
| `initial_creation_time` | timestamp | `2025-12-16T10:00:00Z` | Before restart |
| `final_creation_time` | timestamp | `2025-12-16T10:01:30Z` | After restart |
| `restart_event_found` | bool | `true` | WatchdogRestart event exists |
| `event_mount_path` | string | `/mnt/test` | From event message |

### Test Result State

| State | Exit Code | Meaning |
|-------|-----------|---------|
| `PASS` | 0 | All verifications succeeded |
| `FAIL_RESTART` | 1 | Pod did not restart in time |
| `FAIL_EVENT` | 2 | WatchdogRestart event not found |
| `FAIL_SETUP` | 3 | Cluster/deployment setup failed |
| `FAIL_CLEANUP` | 4 | Cleanup failed (warning only) |

---

## Mock K8sClient State

State managed by mock during unit tests:

```go
type MockK8sClientState struct {
    DeletePodCalls     []string      // Pod names passed to DeletePod
    DeletePodErrors    []error       // Errors to return (queue)
    IsPodTerminating   bool          // Fixed return value
    CanDeletePods      bool          // Fixed return value
    EventsCreated      []RestartEvent // Events passed to CreateEvent
}
```

### Test Scenario Configurations

| Scenario | DeletePodErrors | IsPodTerminating | CanDeletePods |
|----------|-----------------|------------------|---------------|
| Happy path | `[]` | `false` | `true` |
| Retry success | `[err, err, nil]` | `false` | `true` |
| Retry exhausted | `[err, err, err, err]` | `false` | `true` |
| Already terminating | `[]` | `true` | `true` |
| RBAC failure | `[]` | `false` | `false` |

---

## No Persistent Data

This feature does not:
- Create new database schemas
- Store persistent state
- Modify existing data structures

All state is ephemeral:
- Test cluster is deleted after test
- Test results are stdout/exit code
- Mock state exists only during test execution
