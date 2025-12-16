# Quickstart: Pod Restart Watchdog

This guide walks you through enabling and testing the watchdog feature that triggers pod-level restarts when mount health checks fail.

## Prerequisites

- Kubernetes cluster (KIND, minikube, or production)
- `kubectl` configured to access the cluster
- Docker (for building images)
- Go 1.21+ (for local development)

## Quick Setup (KIND)

### 1. Create KIND Cluster

```bash
make kind-create
```

### 2. Build and Deploy with Watchdog Enabled

```bash
# Build the image
make docker-build

# Load into KIND
make kind-load

# Deploy with watchdog configuration
kubectl apply -f deploy/kind/namespace.yaml
kubectl apply -f deploy/kind/rbac.yaml       # NEW: Required for watchdog
kubectl apply -f deploy/kind/configmap.yaml  # Updated with watchdog config
kubectl apply -f deploy/kind/deployment.yaml # Updated with ServiceAccount
kubectl apply -f deploy/kind/service.yaml
```

### 3. Verify Watchdog is Armed

```bash
# Check logs for "watchdog armed" message
kubectl logs -n mount-monitor-dev -l app=mount-monitor -c mount-monitor | grep watchdog

# Expected output:
# {"time":"...","level":"INFO","msg":"watchdog armed","pod":"test-app-with-monitor-xxxxx","namespace":"mount-monitor-dev"}
```

## Configuration

### Enable Watchdog via ConfigMap

```yaml
# deploy/kind/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mount-monitor-config
  namespace: mount-monitor-dev
data:
  config.json: |
    {
      "checkInterval": "10s",
      "readTimeout": "5s",
      "failureThreshold": 3,  // ⚠️ [007]: Renamed from "debounceThreshold"
      "httpPort": 8080,
      "logLevel": "debug",
      "logFormat": "json",
      "canaryFile": ".health-check",
      "mounts": [
        {
          "name": "test-mount",
          "path": "/mnt/test"
        }
      ],
      "watchdog": {
        "enabled": true,
        "restartDelay": "0s",
        "maxRetries": 3
      }
    }
```

### Enable via Environment Variable

```yaml
# In deployment.yaml
env:
- name: WATCHDOG_ENABLED
  value: "true"
```

## Testing Watchdog Behavior

### Test 1: Simulate Mount Failure → Pod Restart

```bash
# Get pod name
POD=$(kubectl get pods -n mount-monitor-dev -l app=mount-monitor -o jsonpath='{.items[0].metadata.name}')

# Watch events in another terminal
kubectl get events -n mount-monitor-dev -w

# Delete the canary file to simulate mount failure
kubectl exec -n mount-monitor-dev $POD -c main-app -- rm /mnt/test/.health-check

# Watch logs (expect: degraded → unhealthy → restart triggered)
kubectl logs -n mount-monitor-dev $POD -c mount-monitor -f

# Expected sequence:
# 1. "health check failed" (x3 for debounce)
# 2. "mount state changed" new_state=unhealthy
# 3. "watchdog restart triggered"
# 4. Pod terminates and is recreated by Deployment controller
```

### Test 2: Verify All Containers Restart Together

```bash
# Before triggering failure, note container start times
kubectl get pods -n mount-monitor-dev -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{range .status.containerStatuses[*]}  {.name}: {.state.running.startedAt}{"\n"}{end}{end}'

# Trigger failure (as above)
# ...

# After pod restarts, verify both containers have new start times
kubectl get pods -n mount-monitor-dev -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{range .status.containerStatuses[*]}  {.name}: {.state.running.startedAt}{"\n"}{end}{end}'
```

### Test 3: Verify Kubernetes Event Created

```bash
# After watchdog triggers, check events
kubectl get events -n mount-monitor-dev --field-selector reason=WatchdogRestart

# Expected output:
# LAST SEEN   TYPE      REASON             OBJECT                              MESSAGE
# 10s         Warning   WatchdogRestart    pod/test-app-with-monitor-xxxxx     Mount /mnt/test unhealthy, triggering pod restart
```

### Test 4: Verify Recovery Cancels Pending Restart

```bash
# Start watching logs
kubectl logs -n mount-monitor-dev $POD -c mount-monitor -f &

# Delete canary (triggers degraded state)
kubectl exec -n mount-monitor-dev $POD -c main-app -- rm /mnt/test/.health-check

# Quickly recreate canary before 3 failures
kubectl exec -n mount-monitor-dev $POD -c main-app -- touch /mnt/test/.health-check

# Expected: "restart cancelled" in logs, pod NOT restarted
```

## Troubleshooting

### Watchdog Not Arming

**Symptom**: Logs show "watchdog disabled" instead of "watchdog armed"

**Check RBAC**:
```bash
# Verify ServiceAccount exists
kubectl get sa mount-monitor -n mount-monitor-dev

# Verify Role and RoleBinding
kubectl get role mount-monitor-watchdog -n mount-monitor-dev
kubectl get rolebinding mount-monitor-watchdog -n mount-monitor-dev

# Check if SA can delete pods
kubectl auth can-i delete pods --as=system:serviceaccount:mount-monitor-dev:mount-monitor -n mount-monitor-dev
# Expected: yes
```

**Check Configuration**:
```bash
# Verify watchdog.enabled is true
kubectl get configmap mount-monitor-config -n mount-monitor-dev -o jsonpath='{.data.config\.json}' | jq .watchdog
```

### Pod Not Restarting

**Symptom**: Mount becomes unhealthy but pod doesn't restart

**Check logs for errors**:
```bash
kubectl logs -n mount-monitor-dev $POD -c mount-monitor | grep -E "(error|failed|denied)"
```

**Common causes**:
1. RBAC permissions missing → Apply `rbac.yaml`
2. ServiceAccount not referenced → Check `spec.serviceAccountName` in deployment
3. Watchdog disabled → Check config

### API Errors

**Symptom**: Logs show "pod deletion failed" with 401/403

```bash
# Re-apply RBAC resources
kubectl apply -f deploy/kind/rbac.yaml

# Restart pod to pick up new ServiceAccount token
kubectl rollout restart deployment/test-app-with-monitor -n mount-monitor-dev
```

## Local Development (Outside Kubernetes)

When running outside Kubernetes, watchdog mode automatically disables:

```bash
# Run locally
go run ./cmd/mount-monitor --config config.json

# Expected log:
# {"level":"INFO","msg":"watchdog disabled","reason":"not running in kubernetes"}
```

The service continues to operate normally with HTTP health endpoints - only pod deletion is disabled.

## Next Steps

- **Production deployment**: Use proper namespace, resource limits, and monitoring
- **Alerting**: Set up alerts on `WatchdogRestart` Kubernetes events
- **Metrics**: Consider adding Prometheus metrics for restart counts (future feature)
