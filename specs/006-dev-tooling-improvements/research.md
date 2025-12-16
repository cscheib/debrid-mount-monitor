# Research: Development Tooling Improvements

**Feature**: 006-dev-tooling-improvements
**Date**: 2025-12-16

## Overview

This research documents findings for implementing automated KIND testing, comprehensive watchdog unit tests, troubleshooting documentation, and namespace customization.

---

## R1: Shell-Based E2E Test Orchestration

### Decision
Use a bash script (`scripts/kind-e2e-test.sh`) orchestrated by a Makefile target for e2e testing.

### Rationale
- **No external dependencies**: Bash is universally available on target platforms (Linux, macOS)
- **Native kubectl/kind integration**: Shell provides direct command execution without wrappers
- **Timeout handling**: Built-in `timeout` command or polling loops for test timing
- **Exit code propagation**: Natural shell exit codes for CI integration

### Alternatives Considered

| Alternative | Rejected Because |
|-------------|------------------|
| Go test with client-go | Would add kubernetes/client-go dependency (~50+ transitive deps), violating Minimal Dependencies principle |
| Python/pytest | Adds Python runtime dependency |
| Makefile-only | Too verbose for complex logic (retries, timeouts, conditional cleanup) |
| Bats (Bash testing) | Additional tooling dependency, overkill for single test script |

### Best Practices Identified
1. **Idempotency**: Script should clean up partial state on re-run
2. **Trap handlers**: Use `trap cleanup EXIT` for reliable cleanup
3. **Polling over sleep**: Use `kubectl wait` or polling loops instead of fixed `sleep` durations
4. **Colored output**: Use ANSI colors for pass/fail visibility (with fallback for non-TTY)
5. **KEEP_CLUSTER flag**: Allow debugging by preserving cluster on failure

---

## R2: Mock K8sClient for Unit Testing

### Decision
Create an interface-based mock in the test file that implements K8sClient behavior.

### Rationale
- **Interface already exists**: The watchdog package already uses interfaces (WatchdogNotifier)
- **Test isolation**: Mock allows testing all code paths without real K8s cluster
- **Error injection**: Mock can simulate API failures, timeouts, permission errors
- **No production changes**: Mock lives in test file only

### Implementation Approach

```go
// In tests/unit/watchdog_test.go
type MockK8sClient struct {
    DeletePodFunc       func(ctx context.Context, name string) error
    IsPodTerminatingFunc func(ctx context.Context, name string) (bool, error)
    CanDeletePodsFunc    func(ctx context.Context) (bool, error)
    CreateEventFunc      func(ctx context.Context, event watchdog.RestartEvent) error
}

func (m *MockK8sClient) DeletePod(ctx context.Context, name string) error {
    if m.DeletePodFunc != nil {
        return m.DeletePodFunc(ctx, name)
    }
    return nil
}
// ... other methods
```

### Test Scenarios Enabled
1. **DeletePod failure → retry logic**: Mock returns error N times, then success
2. **RBAC validation failure**: CanDeletePods returns false
3. **Pod already terminating**: IsPodTerminating returns true
4. **Event creation failure**: CreateEvent returns error (should not block restart)

---

## R3: Namespace Customization Pattern

### Decision
Use `KIND_NAMESPACE` environment variable in Makefile, defaulting to `mount-monitor-dev`.

### Rationale
- **Consistent pattern**: Already using `KIND_CLUSTER_NAME` env var
- **Parallel runs**: Different CI jobs can use different namespaces
- **No code changes**: Only Makefile and deployment scripts affected

### Implementation Details

```makefile
# In Makefile
KIND_NAMESPACE ?= mount-monitor-dev

kind-deploy:
    @kubectl create namespace $(KIND_NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
    @kubectl apply -n $(KIND_NAMESPACE) -f deploy/kind/
```

### Namespace Creation Strategy
- Use `kubectl create namespace ... --dry-run=client -o yaml | kubectl apply -f -` for idempotency
- Creates if not exists, no-op if exists
- Avoids error on re-runs

---

## R4: Watchdog Test Coverage Gaps

### Current State Analysis
Reviewed `tests/unit/watchdog_test.go` - current tests cover:
- ✅ Disabled by config
- ✅ Disabled outside Kubernetes
- ✅ State machine basics (disabled state)
- ✅ WatchdogStatus.String()
- ✅ Config validation
- ✅ Exit function override
- ✅ RestartEvent fields
- ✅ WatchdogState fields

### Coverage Gaps Identified
1. **Armed state transitions**: No test for Disabled → Armed on K8s cluster detection
2. **PendingRestart state**: No test for Armed → PendingRestart on unhealthy
3. **Triggered state**: No test for PendingRestart → Triggered after delay
4. **Retry logic**: No test for DeletePod failure → exponential backoff
5. **Recovery cancellation**: No test for PendingRestart → Armed on healthy
6. **RBAC failure**: No test for CanDeletePods returning false

### Test Strategy
- Create mock K8sClient to simulate in-cluster behavior
- Use short delays (1ms) for timing-sensitive tests
- Test each state transition explicitly
- Verify retry count and backoff timing

---

## R5: Troubleshooting Content Structure

### Decision
Create `docs/troubleshooting.md` with symptom-based organization.

### Rationale
- **Operator-focused**: Organized by what they observe, not by code structure
- **Copy-paste commands**: Diagnostic commands ready to run
- **Progressive disclosure**: Start with common issues, advanced at end

### Content Structure

```markdown
# Troubleshooting Guide

## Quick Diagnostics
- How to check watchdog status
- How to view watchdog logs

## Common Issues

### Pod Not Restarting After Mount Failure
- Symptoms: ...
- Diagnostic commands: ...
- Resolution steps: ...

### RBAC Permission Errors
- Symptoms: ...
- Diagnostic commands: ...
- Resolution steps: ...

### Missing POD_NAME/POD_NAMESPACE
- Symptoms: ...
- Diagnostic commands: ...
- Resolution steps: ...

### Mount Never Detected as Unhealthy
- Symptoms: ...
- Diagnostic commands: ...
- Resolution steps: ...

## Advanced Troubleshooting
- Enabling debug logging
- Inspecting Kubernetes events
- Checking service account token
```

---

## R6: E2E Test Verification Strategy

### Decision
Verify pod restart by checking pod creation timestamp changes and WatchdogRestart event.

### Verification Methods

| Verification | Command | Success Criteria |
|--------------|---------|------------------|
| Pod restart | `kubectl get pod -o jsonpath='{.metadata.creationTimestamp}'` | Timestamp changed |
| All containers restarted | Compare restart counts before/after | Both containers have new start times |
| WatchdogRestart event | `kubectl get events --field-selector reason=WatchdogRestart` | Event exists with correct mount path |
| Canary file removed | `kubectl exec ... -- test -f /mnt/test/.health-check` | File does not exist |

### Timing Considerations
- Watchdog restart delay: 5s (test config)
- K8s API latency: ~1-2s
- Pod termination grace period: 30s default (reduce in test)
- Total test window: ~60-90s for restart cycle

---

## Summary

| Research Item | Decision | Key Insight |
|---------------|----------|-------------|
| R1: E2E Orchestration | Shell script | Bash + kubectl is sufficient, no external deps |
| R2: Mock K8sClient | Interface mock | Enables testing all error paths |
| R3: Namespace Customization | KIND_NAMESPACE env var | Follows existing pattern |
| R4: Test Coverage | 6 gaps identified | Focus on state transitions and retry logic |
| R5: Troubleshooting Docs | Symptom-based structure | Operator-friendly organization |
| R6: E2E Verification | Timestamp + events | Multiple verification points for confidence |

All NEEDS CLARIFICATION items resolved. Ready for Phase 1.
