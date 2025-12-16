# Quickstart: KIND Local Development

**Feature**: 004-kind-local-dev | **Date**: 2025-12-15

Get the mount-monitor sidecar running in a local Kubernetes cluster in under 5 minutes.

---

## Prerequisites

Before starting, ensure you have:

| Tool | Version | Install |
|------|---------|---------|
| Docker | 20.10+ | [docs.docker.com](https://docs.docker.com/get-docker/) |
| kubectl | 1.28+ | [kubernetes.io](https://kubernetes.io/docs/tasks/tools/) |
| KIND | 0.20+ | `brew install kind` or [kind.sigs.k8s.io](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) |
| Go | 1.21+ | Required only for building from source |

**Verify installation:**
```bash
docker --version
kubectl version --client
kind --version
```

---

## Quick Start (3 Commands)

```bash
# 1. Create KIND cluster and build image
make kind-create kind-load

# 2. Deploy the monitor sidecar
make kind-deploy

# 3. Watch the logs
make kind-logs
```

That's it! You should see the monitor checking `/mnt/test` every 10 seconds.

---

## Step-by-Step Workflow

### 1. Create the KIND Cluster

```bash
make kind-create
```

This creates a single-node Kubernetes cluster named `debrid-mount-monitor` (customizable via `KIND_CLUSTER_NAME` env var).

**Verify cluster is running:**
```bash
kubectl cluster-info
kubectl get nodes
```

### 2. Build and Load the Monitor Image

```bash
make kind-load
```

This:
1. Builds the Docker image (`mount-monitor:dev`)
2. Loads it into the KIND cluster

**Verify image is loaded:**
```bash
docker exec -it debrid-mount-monitor-control-plane crictl images | grep mount-monitor
```

### 3. Deploy the Monitor

```bash
make kind-deploy
```

This applies the Kubernetes manifests:
- Namespace: `mount-monitor-dev`
- ConfigMap: `mount-monitor-config`
- Deployment: `test-app-with-monitor` (main-app + mount-monitor sidecar)

**Verify deployment:**
```bash
kubectl -n mount-monitor-dev get pods
kubectl -n mount-monitor-dev get pods -o wide
```

### 4. View Monitor Logs

```bash
make kind-logs
```

Or manually:
```bash
kubectl -n mount-monitor-dev logs -l app=test-app-with-monitor -c mount-monitor -f
```

### 5. Check Health Endpoints

**From within the cluster:**
```bash
# Get pod name
POD=$(kubectl -n mount-monitor-dev get pod -l app=test-app-with-monitor -o jsonpath='{.items[0].metadata.name}')

# Check liveness
kubectl -n mount-monitor-dev exec -it $POD -c main-app -- wget -qO- http://localhost:8080/healthz/live

# Check readiness
kubectl -n mount-monitor-dev exec -it $POD -c main-app -- wget -qO- http://localhost:8080/healthz/ready

# Check detailed status
kubectl -n mount-monitor-dev exec -it $POD -c main-app -- wget -qO- http://localhost:8080/healthz/status
```

---

## Simulating Mount Failures

The monitor watches `/mnt/test/.health-check`. Removing this file simulates a mount failure.

### Trigger a Failure

```bash
# Get pod name
POD=$(kubectl -n mount-monitor-dev get pod -l app=test-app-with-monitor -o jsonpath='{.items[0].metadata.name}')

# Remove the canary file (simulates mount failure)
kubectl -n mount-monitor-dev exec -it $POD -c main-app -- rm /mnt/test/.health-check

# Watch the logs - should see health check failures
make kind-logs
```

**Expected behavior:**
1. Monitor detects missing canary file
2. After 3 consecutive failures (debounce threshold), mount marked UNHEALTHY
3. Readiness probe starts failing (HTTP 503)
4. Pod status changes to `Not Ready`

**Verify readiness failure:**
```bash
kubectl -n mount-monitor-dev get pods
# STATUS should show 1/2 Ready (main-app ready, mount-monitor not ready)
```

### Restore Health

```bash
# Recreate the canary file
kubectl -n mount-monitor-dev exec -it $POD -c main-app -- sh -c 'echo "healthy" > /mnt/test/.health-check'

# Watch recovery in logs
make kind-logs
```

**Expected behavior:**
1. Monitor detects canary file restored
2. Mount marked HEALTHY
3. Readiness probe returns success (HTTP 200)
4. Pod status returns to `Ready`

---

## Development Iteration

### Rebuild and Redeploy After Code Changes

```bash
make kind-redeploy
```

This single command:
1. Rebuilds the Docker image
2. Loads it into KIND
3. Restarts the deployment with the new image

**Iteration cycle time:** ~30-60 seconds

### View All Container Logs

```bash
# Both containers
kubectl -n mount-monitor-dev logs $POD --all-containers=true -f

# Just main-app
kubectl -n mount-monitor-dev logs $POD -c main-app

# Just mount-monitor
kubectl -n mount-monitor-dev logs $POD -c mount-monitor
```

### Exec Into Containers

```bash
# Shell into main-app (has the mount)
kubectl -n mount-monitor-dev exec -it $POD -c main-app -- sh

# Check mount contents
ls -la /mnt/test/
```

---

## Cleanup

### Delete the Deployment (Keep Cluster)

```bash
kubectl delete namespace mount-monitor-dev
```

### Delete Everything (Cluster + Resources)

```bash
make kind-delete
```

---

## Troubleshooting

### Pod Stuck in ImagePullBackOff

**Cause:** Image not loaded into KIND, or using wrong tag.

**Fix:**
```bash
# Verify image is loaded
docker exec -it debrid-mount-monitor-control-plane crictl images | grep mount-monitor

# Reload image
make kind-load

# Restart deployment
kubectl -n mount-monitor-dev rollout restart deployment/test-app-with-monitor
```

### Pod Stuck in Pending

**Cause:** Cluster resources exhausted or node not ready.

**Fix:**
```bash
kubectl describe pod -n mount-monitor-dev
# Check Events section for scheduling issues

# Verify node is ready
kubectl get nodes
```

### Health Endpoint Returns 503 Immediately

**Cause:** Canary file doesn't exist (init container may have failed).

**Fix:**
```bash
# Check init container logs
kubectl -n mount-monitor-dev logs $POD -c init-canary

# Manually create canary
kubectl -n mount-monitor-dev exec -it $POD -c main-app -- sh -c 'echo "healthy" > /mnt/test/.health-check'
```

### KIND Cluster Won't Delete

**Cause:** Docker containers in bad state.

**Fix:**
```bash
# Force delete
kind delete cluster --name debrid-mount-monitor

# If that fails, remove Docker containers manually
docker ps -a | grep kind | awk '{print $1}' | xargs docker rm -f
```

---

## Makefile Targets Reference

| Target | Description |
|--------|-------------|
| `kind-create` | Create KIND cluster |
| `kind-delete` | Delete KIND cluster |
| `kind-load` | Build image and load into KIND |
| `kind-deploy` | Apply Kubernetes manifests |
| `kind-redeploy` | Rebuild, reload, and restart |
| `kind-logs` | Tail mount-monitor logs |
| `kind-status` | Show pod and deployment status |

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KIND_CLUSTER_NAME` | `debrid-mount-monitor` | Cluster name for KIND |

**Example: Custom cluster name**
```bash
KIND_CLUSTER_NAME=my-cluster make kind-create
```
