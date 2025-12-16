# Mount Monitor Troubleshooting Guide

This guide helps operators diagnose and resolve common issues with the mount health monitor and watchdog sidecar.

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Issue: Pod Not Restarting After Mount Failure](#issue-pod-not-restarting-after-mount-failure)
- [Issue: RBAC Permission Errors](#issue-rbac-permission-errors)
- [Issue: Missing POD_NAME/POD_NAMESPACE](#issue-missing-pod_namepod_namespace)
- [Issue: Mount Never Detected as Unhealthy](#issue-mount-never-detected-as-unhealthy)
- [Advanced Troubleshooting](#advanced-troubleshooting)
  - [Enable Debug Logging](#enable-debug-logging)
  - [Check Kubernetes Events](#check-kubernetes-events)
  - [Inspect ServiceAccount Token](#inspect-serviceaccount-token)
  - [Test API Connectivity](#test-api-connectivity)
  - [Verify Metrics Endpoint](#verify-metrics-endpoint)
  - [Manual Mount Failure Simulation](#manual-mount-failure-simulation)
- [Common Log Messages Reference](#common-log-messages-reference)
- [Getting Help](#getting-help)

---

## Quick Diagnostics

Run these commands first to get an overview of your deployment status:

```bash
# Check pod status
kubectl -n <namespace> get pods -l app=<your-app>

# Check mount-monitor logs
kubectl -n <namespace> logs <pod-name> -c mount-monitor --tail=50

# Check Kubernetes events for the pod
kubectl -n <namespace> get events --field-selector involvedObject.name=<pod-name>

# Check watchdog status via metrics endpoint
kubectl -n <namespace> exec <pod-name> -c mount-monitor -- wget -qO- http://localhost:8080/metrics
```

---

## Issue: Pod Not Restarting After Mount Failure

### Symptoms
- Mount becomes unhealthy (visible in logs or `/healthz/ready` returns 503)
- Pod continues running instead of being deleted/restarted
- No `WatchdogRestart` events in Kubernetes events

### Diagnostics

1. **Check if watchdog is enabled**:
   ```bash
   kubectl -n <namespace> logs <pod-name> -c mount-monitor | grep -i "watchdog"
   ```
   Look for: `watchdog armed` (working) or `watchdog disabled` (not working)

2. **Check watchdog configuration**:
   ```bash
   kubectl -n <namespace> get configmap mount-monitor-config -o yaml
   ```
   Verify `watchdog.enabled: true` in the config.

3. **Check if pod has required environment variables**:
   ```bash
   kubectl -n <namespace> get pod <pod-name> -o yaml | grep -A 2 "POD_NAME\|POD_NAMESPACE"
   ```

4. **Check if mount is actually unhealthy**:
   ```bash
   kubectl -n <namespace> exec <pod-name> -c mount-monitor -- wget -qO- http://localhost:8080/healthz/ready
   # Returns 503 if unhealthy, 200 if healthy
   ```

### Resolution

1. **If watchdog shows "disabled by configuration"**:
   Update ConfigMap to enable watchdog:
   ```yaml
   watchdog:
     enabled: true
     restartDelay: "0s"  # or desired delay
     maxRetries: 3
   ```

2. **If watchdog shows "not in cluster"**:
   The monitor is not running inside Kubernetes. Watchdog only works in-cluster.

3. **If POD_NAME or POD_NAMESPACE is missing**:
   See "Missing POD_NAME/POD_NAMESPACE" section below.

4. **If RBAC errors appear**:
   See "RBAC Permission Errors" section below.

---

## Issue: RBAC Permission Errors

### Symptoms
- Log message: `watchdog disabled reason=rbac_missing`
- Log message: `watchdog disabled reason=rbac_check_failed`
- No pod restarts occurring despite unhealthy mounts

### Diagnostics

1. **Check ServiceAccount configuration**:
   ```bash
   kubectl -n <namespace> get pod <pod-name> -o yaml | grep serviceAccountName
   ```

2. **Check if ServiceAccount exists**:
   ```bash
   kubectl -n <namespace> get serviceaccount mount-monitor
   ```

3. **Check Role/RoleBinding**:
   ```bash
   kubectl -n <namespace> get role,rolebinding | grep mount-monitor
   ```

4. **Test permissions manually**:
   ```bash
   kubectl auth can-i delete pods --as=system:serviceaccount:<namespace>:mount-monitor -n <namespace>
   ```

### Resolution

1. **Create the required RBAC resources**:
   ```yaml
   apiVersion: v1
   kind: ServiceAccount
   metadata:
     name: mount-monitor
     namespace: <namespace>
   ---
   apiVersion: rbac.authorization.k8s.io/v1
   kind: Role
   metadata:
     name: mount-monitor
     namespace: <namespace>
   rules:
   - apiGroups: [""]
     resources: ["pods"]
     verbs: ["get", "delete"]
   - apiGroups: [""]
     resources: ["events"]
     verbs: ["create"]
   ---
   apiVersion: rbac.authorization.k8s.io/v1
   kind: RoleBinding
   metadata:
     name: mount-monitor
     namespace: <namespace>
   subjects:
   - kind: ServiceAccount
     name: mount-monitor
     namespace: <namespace>
   roleRef:
     kind: Role
     name: mount-monitor
     apiGroup: rbac.authorization.k8s.io
   ```

2. **Update deployment to use ServiceAccount**:
   ```yaml
   spec:
     template:
       spec:
         serviceAccountName: mount-monitor
   ```

---

## Issue: Missing POD_NAME/POD_NAMESPACE

### Symptoms
- Log message: `watchdog disabled reason=missing_pod_info`
- Watchdog fails to identify which pod to delete

### Diagnostics

1. **Check environment variables in pod spec**:
   ```bash
   kubectl -n <namespace> get pod <pod-name> -o yaml | grep -A 5 "env:"
   ```

### Resolution

Add the required environment variables using the Downward API:

```yaml
spec:
  containers:
  - name: mount-monitor
    env:
    - name: POD_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
    - name: POD_NAMESPACE
      valueFrom:
        fieldRef:
          fieldPath: metadata.namespace
```

---

## Issue: Mount Never Detected as Unhealthy

### Symptoms
- Physical mount is clearly broken/stale
- `/healthz/ready` always returns 200
- Logs show "mount healthy" when it shouldn't be

### Diagnostics

1. **Check what paths are being monitored**:
   ```bash
   kubectl -n <namespace> logs <pod-name> -c mount-monitor | grep "monitoring mount"
   ```

2. **Check canary file existence**:
   ```bash
   kubectl -n <namespace> exec <pod-name> -c mount-monitor -- ls -la /mnt/your-mount/.health-check
   ```

3. **Check mount accessibility from monitor container**:
   ```bash
   kubectl -n <namespace> exec <pod-name> -c mount-monitor -- cat /mnt/your-mount/.health-check
   ```

4. **Verify volume mount configuration**:
   ```bash
   kubectl -n <namespace> get pod <pod-name> -o yaml | grep -A 10 "volumeMounts"
   ```

### Resolution

1. **If canary file doesn't exist**:
   Create the canary file (this should be done by your main application or init container):
   ```bash
   kubectl -n <namespace> exec <pod-name> -c main-app -- sh -c 'echo healthy > /mnt/your-mount/.health-check'
   ```

2. **If mount paths don't match**:
   Verify ConfigMap paths match the actual volume mounts in your pod spec.

3. **If monitor can't access the mount**:
   Ensure the mount-monitor container has the volume mounted:
   ```yaml
   containers:
   - name: mount-monitor
     volumeMounts:
     - name: your-volume
       mountPath: /mnt/your-mount
       readOnly: true  # Monitor only needs read access
   ```

4. **If using a different canary filename**:
   Update ConfigMap to match:
   ```yaml
   mounts:
   - name: your-mount
     path: /mnt/your-mount
     canaryFile: ".your-custom-canary"  # Default is .health-check
   ```

---

## Advanced Troubleshooting

### Enable Debug Logging

Set `logLevel` to `debug` in your ConfigMap:

```yaml
data:
  config.json: |
    {
      "logLevel": "debug",
      ...
    }
```

Debug output includes:
- Individual health check results
- State machine transitions
- API call details

### Check Kubernetes Events

View all events related to the pod:

```bash
# Recent events
kubectl -n <namespace> get events --field-selector involvedObject.name=<pod-name> --sort-by='.lastTimestamp'

# Watch for new events
kubectl -n <namespace> get events -w --field-selector involvedObject.name=<pod-name>
```

Look for:
- `WatchdogRestart` - Pod restart triggered by watchdog
- `FailedMount` - Volume mount issues
- `Unhealthy` - Liveness/readiness probe failures

### Inspect ServiceAccount Token

Verify the ServiceAccount token is mounted correctly:

```bash
# Check token exists
kubectl -n <namespace> exec <pod-name> -c mount-monitor -- ls -la /var/run/secrets/kubernetes.io/serviceaccount/

# Check token contents (first 50 chars only for security)
kubectl -n <namespace> exec <pod-name> -c mount-monitor -- sh -c 'head -c 50 /var/run/secrets/kubernetes.io/serviceaccount/token && echo "..."'

# Check namespace file
kubectl -n <namespace> exec <pod-name> -c mount-monitor -- cat /var/run/secrets/kubernetes.io/serviceaccount/namespace
```

### Test API Connectivity

Test if the monitor can reach the Kubernetes API:

```bash
kubectl -n <namespace> exec <pod-name> -c mount-monitor -- sh -c '
  TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
  # WARNING: -k disables certificate verification. Safe here because we are
  # connecting to the in-cluster Kubernetes API using the service account token.
  # Do NOT use -k when connecting to external APIs.
  curl -sk -H "Authorization: Bearer $TOKEN" \
    https://kubernetes.default.svc/api/v1/namespaces/$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace)/pods
'
```

### Verify Metrics Endpoint

Check internal metrics for debugging:

```bash
# Port-forward to metrics endpoint
kubectl -n <namespace> port-forward <pod-name> 8080:8080

# In another terminal:
curl http://localhost:8080/metrics
curl http://localhost:8080/healthz/ready
curl http://localhost:8080/healthz/live
```

### Manual Mount Failure Simulation

Test the watchdog by simulating a mount failure:

```bash
# Get pod name
POD=$(kubectl -n <namespace> get pod -l app=<your-app> -o name)

# Remove canary file to simulate failure
kubectl -n <namespace> exec $POD -c main-app -- rm /mnt/your-mount/.health-check

# Watch logs to see state transition
kubectl -n <namespace> logs $POD -c mount-monitor -f

# Watch for pod restart
kubectl -n <namespace> get pods -w
```

---

## Common Log Messages Reference

| Log Message | Meaning | Action |
|-------------|---------|--------|
| `watchdog armed` | Watchdog is active and monitoring | Normal - no action needed |
| `watchdog disabled reason=not_in_cluster` | Running outside Kubernetes | Expected if running locally |
| `watchdog disabled reason=rbac_missing` | Missing pod delete permission | Check RBAC configuration |
| `watchdog disabled reason=k8s_client_error` | Can't create K8s client | Check ServiceAccount token |
| `mount unhealthy` | Canary file check failed | Check mount status |
| `watchdog restart pending` | Restart delay countdown started | Pod will restart after delay |
| `watchdog restart triggered` | Pod deletion initiated | Pod should restart soon |
| `watchdog restart cancelled` | Mount recovered before restart | Normal recovery behavior |
| `pod deletion successful` | K8s API accepted delete request | Pod will terminate |
| `pod deletion failed` | K8s API rejected delete request | Check RBAC and logs |

---

## Getting Help

If you've tried the above steps and still have issues:

1. Collect debug logs: `kubectl -n <namespace> logs <pod-name> -c mount-monitor > mount-monitor.log`
2. Collect pod description: `kubectl -n <namespace> describe pod <pod-name> > pod-describe.txt`
3. Collect events: `kubectl -n <namespace> get events > events.txt`
4. Open an issue at: https://github.com/cscheib/debrid-mount-monitor/issues
