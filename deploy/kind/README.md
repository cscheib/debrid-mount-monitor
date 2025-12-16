# KIND Local Development Environment

This directory contains configuration files for running the mount-monitor sidecar in a local Kubernetes cluster using KIND (Kubernetes IN Docker).

## Prerequisites

Before starting, ensure you have the following tools installed:

| Tool | Minimum Version | Installation |
|------|-----------------|--------------|
| Docker | 20.10+ | [docs.docker.com](https://docs.docker.com/get-docker/) |
| kubectl | 1.28+ | [kubernetes.io](https://kubernetes.io/docs/tasks/tools/) |
| KIND | 0.20+ | `brew install kind` or [kind.sigs.k8s.io](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) |

Verify your installation:

```bash
docker --version
kubectl version --client
kind --version
```

## Quick Start

```bash
# 1. Create the KIND cluster
make kind-create

# 2. Build and load the monitor image
make kind-load

# 3. Deploy the monitor sidecar
make kind-deploy

# 4. View logs
make kind-logs
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `kind-create` | Create a local KIND cluster |
| `kind-delete` | Delete the KIND cluster |
| `kind-status` | Show cluster and pod status |
| `kind-load` | Build Docker image and load into KIND |
| `kind-deploy` | Deploy the monitor to the cluster |
| `kind-undeploy` | Remove the deployment |
| `kind-logs` | Tail the mount-monitor container logs |
| `kind-redeploy` | Rebuild, reload, and restart (quick iteration) |
| `kind-help` | Show workflow help |

## Quick Iteration Workflow

After making code changes, use `kind-redeploy` for rapid iteration:

```bash
# Make your code changes, then:
make kind-redeploy
```

This single command will:
1. Rebuild the Docker image
2. Load it into the KIND cluster
3. Restart the deployment with the new image
4. Wait for the rollout to complete

**Target iteration time**: Under 60 seconds from code change to running pod.

## Testing Health Endpoints

A NodePort service exposes the monitor's health endpoints on port 30080, allowing you to test directly from your host machine:

```bash
# Check if monitor process is alive
curl localhost:30080/healthz/live

# Check if mounts are healthy (used by Kubernetes readiness probe)
curl localhost:30080/healthz/ready

# Get detailed status of all monitored mounts
curl localhost:30080/healthz/status | jq
```

This is useful for:
- Debugging health check issues without kubectl exec
- Scripting automated tests against the health endpoints
- Monitoring mount status during failure simulation

## Simulating Mount Failures

The deployment creates a simulated mount at `/mnt/test` with a canary file `.health-check`. You can manually simulate mount failures to test the monitor's behavior.

### Trigger a Mount Failure

```bash
# Get the pod name
POD=$(kubectl -n mount-monitor-dev get pod -l app=test-app-with-monitor -o jsonpath='{.items[0].metadata.name}')

# Remove the canary file to simulate a mount failure
kubectl -n mount-monitor-dev exec $POD -c main-app -- rm /mnt/test/.health-check
```

### Verifying Probe Behavior

After removing the canary file:

> **⏱️ Timing Note:** The monitor's internal state and Kubernetes probe timing are intentionally different:
> - **Monitor internal**: 10s check interval × 3 failures = ~30s until UNHEALTHY state
> - **Kubernetes probe**: 5s period × 3 failures = ~15s until pod NotReady
>
> This means the pod may show `1/2 Ready` before the monitor logs show "UNHEALTHY". This is expected—Kubernetes detects the probe failures faster than the monitor's debounce threshold.

1. **Watch the logs** to see health check failures:
   ```bash
   make kind-logs
   ```

2. **Check pod status** - after 3 consecutive failures (debounce threshold), the pod becomes NotReady:
   ```bash
   kubectl -n mount-monitor-dev get pods -w
   ```

   Expected output progression:
   ```
   NAME                                    READY   STATUS    RESTARTS   AGE
   test-app-with-monitor-xxxxx             2/2     Running   0          5m
   test-app-with-monitor-xxxxx             1/2     Running   0          5m    # After ~30s
   ```

3. **Check readiness probe status**:
   ```bash
   kubectl -n mount-monitor-dev describe pod $POD | grep -A5 "Readiness:"
   ```

4. **View events** to see probe failures and potential restarts:
   ```bash
   kubectl -n mount-monitor-dev get events --sort-by='.lastTimestamp' | tail -20
   ```

### Restoring Mount Health

```bash
# Recreate the canary file
kubectl -n mount-monitor-dev exec $POD -c main-app -- sh -c 'echo "healthy" > /mnt/test/.health-check'
```

After restoration:
- The next health check will pass
- Pod status returns to `2/2 Ready`
- Logs will show state transition back to HEALTHY

### Complete Failure Simulation Walkthrough

```bash
# 1. Ensure deployment is healthy
make kind-status

# 2. Open a second terminal and watch logs
make kind-logs

# 3. In the first terminal, trigger failure
POD=$(kubectl -n mount-monitor-dev get pod -l app=test-app-with-monitor -o name)
kubectl -n mount-monitor-dev exec $POD -c main-app -- rm /mnt/test/.health-check

# 4. Watch the logs - you'll see:
#    - Health check failures every 10 seconds
#    - State transition to DEGRADED after first failure
#    - State transition to UNHEALTHY after 3 failures
#    - Readiness probe starts failing

# 5. Check pod status in another terminal
kubectl -n mount-monitor-dev get pods -w

# 6. Restore health
kubectl -n mount-monitor-dev exec $POD -c main-app -- sh -c 'echo "healthy" > /mnt/test/.health-check'

# 7. Watch recovery in logs and pod status returning to Ready
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KIND_CLUSTER_NAME` | `debrid-mount-monitor` | Name of the KIND cluster |

### Using a Custom Cluster Name

```bash
# Create cluster with custom name
KIND_CLUSTER_NAME=my-cluster make kind-create

# All subsequent commands use the same name
KIND_CLUSTER_NAME=my-cluster make kind-load kind-deploy
KIND_CLUSTER_NAME=my-cluster make kind-logs
KIND_CLUSTER_NAME=my-cluster make kind-delete
```

## Troubleshooting

### Pod Stuck in ImagePullBackOff

**Cause**: The Docker image wasn't loaded into KIND, or wrong tag was used.

**Solution**:
```bash
# Verify image is loaded
docker exec -it debrid-mount-monitor-control-plane crictl images | grep mount-monitor

# Reload the image
make kind-load

# Restart the deployment
kubectl -n mount-monitor-dev rollout restart deployment/test-app-with-monitor
```

### Pod Stuck in Pending

**Cause**: Cluster resources exhausted or node not ready.

**Solution**:
```bash
# Check node status
kubectl get nodes

# Check pod events
kubectl -n mount-monitor-dev describe pod -l app=test-app-with-monitor

# If node issues, recreate cluster
make kind-delete kind-create
```

### Health Endpoint Returns 503 Immediately

**Cause**: The canary file doesn't exist (init container may have failed).

**Solution**:
```bash
# Check init container logs
POD=$(kubectl -n mount-monitor-dev get pod -l app=test-app-with-monitor -o name)
kubectl -n mount-monitor-dev logs $POD -c init-canary

# Manually create the canary file
kubectl -n mount-monitor-dev exec $POD -c main-app -- sh -c 'echo "healthy" > /mnt/test/.health-check'
```

### KIND Cluster Won't Delete

**Cause**: Docker containers in a bad state.

**Solution**:
```bash
# Force delete the cluster
kind delete cluster --name debrid-mount-monitor

# If that fails, manually remove containers
docker ps -a | grep kind | awk '{print $1}' | xargs docker rm -f

# Remove KIND networks
docker network ls | grep kind | awk '{print $1}' | xargs docker network rm
```

### Docker Not Running

**Cause**: Docker daemon is not started.

**Solution**:
```bash
# On macOS, start Docker Desktop
open -a Docker

# On Linux, start the service
sudo systemctl start docker
```

### kubectl Cannot Connect to Cluster

**Cause**: kubeconfig not set or cluster not running.

**Solution**:
```bash
# Check if cluster is running
kind get clusters

# Verify kubeconfig is set correctly
kubectl config current-context
# Should show: kind-debrid-mount-monitor

# If needed, set the context
kubectl config use-context kind-debrid-mount-monitor
```

## Files in This Directory

| File | Description |
|------|-------------|
| `kind-config.yaml` | KIND cluster configuration (single node, K8s v1.28) |
| `namespace.yaml` | Kubernetes namespace for isolation |
| `configmap.yaml` | JSON config file for mount-monitor (mounted as `/etc/mount-monitor/config.json`) |
| `deployment.yaml` | Sidecar deployment with probes, init container, and config file mount |
| `service.yaml` | NodePort service exposing health endpoints on port 30080 |

### Configuration Approach

The deployment uses a **JSON config file** mounted from a ConfigMap rather than environment variables. This:
- Exercises the JSON config parsing code path
- Demonstrates per-mount configuration with named mounts
- More closely mimics production deployments using config files

The config file is mounted at `/etc/mount-monitor/config.json` and passed via `--config` flag.

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                        KIND Cluster                               │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                Namespace: mount-monitor-dev                 │  │
│  │  ┌──────────────────────────────────────────────────────┐  │  │
│  │  │               Pod: test-app-with-monitor              │  │  │
│  │  │                                                       │  │  │
│  │  │  ┌──────────────┐    ┌────────────────────────────┐  │  │  │
│  │  │  │   main-app   │    │      mount-monitor         │  │  │  │
│  │  │  │  (alpine)    │    │        (sidecar)           │  │  │  │
│  │  │  │              │    │                            │  │  │  │
│  │  │  │  /mnt/test   │◄───│  /mnt/test (readonly)      │  │  │  │
│  │  │  │   (rw)       │    │  /etc/mount-monitor/       │  │  │  │
│  │  │  └──────────────┘    │    config.json (from CM)   │  │  │  │
│  │  │                      │                            │  │  │  │
│  │  │                      │  :8080/healthz/live        │  │  │  │
│  │  │                      │  :8080/healthz/ready       │  │  │  │
│  │  │                      └────────────────────────────┘  │  │  │
│  │  │                               │                      │  │  │
│  │  │               ┌───────────────┘                      │  │  │
│  │  │               ▼                                      │  │  │
│  │  │  ┌──────────────────────────────┐  ┌─────────────┐  │  │  │
│  │  │  │      emptyDir volume         │  │  ConfigMap  │  │  │  │
│  │  │  │  /mnt/test/.health-check     │  │ config.json │  │  │  │
│  │  │  └──────────────────────────────┘  └─────────────┘  │  │  │
│  │  └──────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

## Cleanup

```bash
# Remove deployment only (keep cluster)
make kind-undeploy

# Remove everything
make kind-delete
```
