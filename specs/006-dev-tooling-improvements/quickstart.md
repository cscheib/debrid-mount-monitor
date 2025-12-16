# Quickstart: Development Tooling Improvements

**Feature**: 006-dev-tooling-improvements
**Date**: 2025-12-16

## Prerequisites

- Docker running
- `kind` installed (`brew install kind` or see [KIND docs](https://kind.sigs.k8s.io/docs/user/quick-start/#installation))
- `kubectl` installed
- Go 1.21+

---

## Quick Commands

### Run Automated E2E Watchdog Test

```bash
# Full automated test (creates cluster, tests, cleans up)
make kind-test

# Keep cluster for debugging on failure
KEEP_CLUSTER=1 make kind-test
```

### Run Unit Tests with Coverage

```bash
# Run all tests with coverage
make test-coverage

# View coverage report
open coverage.html

# Run only watchdog tests
go test -v ./tests/unit/... -run TestWatchdog
```

### Deploy to Custom Namespace

```bash
# Create cluster with custom namespace
make kind-create
KIND_NAMESPACE=my-namespace make kind-deploy

# View resources in custom namespace
kubectl -n my-namespace get pods
```

---

## E2E Test Workflow

### What the Test Does

1. **Setup** (30-60s)
   - Creates KIND cluster (or reuses existing)
   - Builds Docker image
   - Loads image into cluster
   - Deploys with test config (short delays)

2. **Test Execution** (60-90s)
   - Waits for pod to be ready
   - Records initial pod creation timestamp
   - Removes canary file to simulate mount failure
   - Waits for watchdog to trigger restart
   - Verifies pod creation timestamp changed
   - Verifies WatchdogRestart event exists

3. **Cleanup**
   - Deletes cluster (unless `KEEP_CLUSTER=1`)
   - Reports pass/fail

### Expected Output (Pass)

```text
=== KIND E2E Test: Watchdog Pod Restart ===

[1/6] Checking prerequisites...
  ✓ Docker is running
  ✓ kind is installed
  ✓ kubectl is installed

[2/6] Setting up cluster...
  ✓ Cluster 'debrid-mount-monitor-test' created

[3/6] Building and deploying...
  ✓ Image built and loaded
  ✓ Deployment ready

[4/6] Simulating mount failure...
  ✓ Canary file removed

[5/6] Verifying restart...
  ✓ Pod restarted (new creation timestamp)
  ✓ WatchdogRestart event found

[6/6] Cleanup...
  ✓ Cluster deleted

=== TEST PASSED ===
```

---

## Unit Test Coverage

### Running Coverage Analysis

```bash
# Generate coverage for watchdog package
go test -v -coverprofile=coverage.out ./tests/unit/...

# Check coverage percentage
go tool cover -func=coverage.out | grep watchdog

# Target: 80%+ coverage for watchdog package
```

### Key Test Scenarios

| Test | Description |
|------|-------------|
| `TestWatchdog_StateTransitions` | Disabled → Armed → PendingRestart → Triggered |
| `TestWatchdog_RetryLogic` | DeletePod fails N times, then succeeds |
| `TestWatchdog_RecoveryCancellation` | Mount recovers during restart delay |
| `TestWatchdog_RBACFailure` | CanDeletePods returns false |
| `TestWatchdog_AlreadyTerminating` | IsPodTerminating returns true |

---

## Troubleshooting

### Test Fails: Pod Not Restarting

```bash
# Check watchdog logs
kubectl -n watchdog-e2e-test logs -l app=test-app-with-monitor -c mount-monitor

# Check if watchdog is enabled
kubectl -n watchdog-e2e-test get configmap mount-monitor-config -o yaml | grep enabled

# Check RBAC permissions
kubectl auth can-i delete pods -n watchdog-e2e-test --as=system:serviceaccount:watchdog-e2e-test:mount-monitor
```

### Test Fails: Cluster Setup

```bash
# Check Docker resources
docker system df

# Delete any stuck clusters
kind delete clusters --all

# Retry
make kind-test
```

### Debugging with KEEP_CLUSTER

```bash
# Run test but keep cluster on failure
KEEP_CLUSTER=1 make kind-test

# Inspect cluster state
kubectl -n watchdog-e2e-test get pods -o wide
kubectl -n watchdog-e2e-test describe pod -l app=test-app-with-monitor
kubectl -n watchdog-e2e-test get events --sort-by='.lastTimestamp'

# Cleanup manually when done
make kind-delete
```

---

## Configuration Reference

### Test ConfigMap Values

| Setting | Value | Why |
|---------|-------|-----|
| `check_interval` | 2s | Fast feedback |
| `failure_threshold` | 2 | Quick failure detection |
| `watchdog.enabled` | true | Required for test |
| `watchdog.restart_delay` | 5s | Time to observe state change |

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KIND_CLUSTER_NAME` | `debrid-mount-monitor` | Cluster name |
| `KIND_NAMESPACE` | `mount-monitor-dev` | Deployment namespace |
| `KEEP_CLUSTER` | `0` | Set to `1` to preserve cluster |

---

## See Also

- [Troubleshooting Guide](../../docs/troubleshooting.md) - Common issues and resolutions
- [KIND README](../../deploy/kind/README.md) - Manual KIND workflow
- [Watchdog Spec](../005-pod-restart-watchdog/spec.md) - Watchdog feature details
