# Research: KIND Local Development Environment

**Feature**: 004-kind-local-dev | **Date**: 2025-12-15

## Summary

Research findings for implementing KIND (Kubernetes IN Docker) local development environment for the mount-monitor sidecar service.

---

## 1. KIND Cluster Configuration

### Decision: Single-Node Cluster
**Rationale**: Single-node is sufficient for sidecar testing, provides faster creation/teardown (<2min target), and minimizes resource usage. Multi-node only needed for scheduling/affinity testing.

**Alternatives Considered**:
- Multi-node cluster: Rejected - adds complexity without benefit for sidecar testing
- Minikube: Rejected - heavier weight, VM-based, slower iteration

### Decision: Kubernetes v1.28+
**Rationale**: Stable release with mature sidecar support. Native sidecar container feature (stable in 1.29+) improves lifecycle management. Wide compatibility with kubectl versions.

**Alternatives Considered**:
- v1.27: Missing native sidecar container improvements
- v1.30+: Bleeding edge, potential instability

### Decision: Customizable Cluster Name via Environment Variable
**Rationale**: Per spec clarification - allows developers to run parallel clusters for different projects. Default: `debrid-mount-monitor`.

**Implementation**:
```bash
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-debrid-mount-monitor}"
kind create cluster --name "$KIND_CLUSTER_NAME"
```

---

## 2. Image Loading Strategy

### Decision: Direct Image Loading with Explicit Tags
**Rationale**: Fastest iteration cycle. Avoids registry round-trips. Specific tags prevent `imagePullPolicy: Always` issues.

**Critical Configuration**:
```yaml
imagePullPolicy: IfNotPresent  # Required for local images
```

**Alternatives Considered**:
- Local registry: Rejected - adds complexity for single-developer use case
- `:latest` tag: Rejected - causes ImagePullBackOff with default pull policy

### Decision: Version Tag Pattern `mount-monitor:dev`
**Rationale**: Fixed `dev` tag simplifies iteration (no version bumping). Clear signal this is local development image.

---

## 3. Sidecar Deployment Pattern

### Decision: Shared emptyDir Volume for Simulated Mounts
**Rationale**: emptyDir is ephemeral and pod-scoped - perfect for simulating mount points. No hostPath dependencies keeps manifests portable.

**Structure**:
```yaml
volumes:
- name: simulated-mount
  emptyDir: {}
```

**Alternatives Considered**:
- hostPath: Rejected - requires KIND extraMounts configuration, less portable
- PersistentVolume: Rejected - overkill for local testing, adds complexity

### Decision: Init Container Creates Canary File
**Rationale**: Ensures mount starts in healthy state. Simpler than pre-populating volumes externally.

```yaml
initContainers:
- name: init-canary
  image: busybox:1.36
  command: ['sh', '-c', 'echo "healthy" > /mnt/test/.health-check']
  volumeMounts:
  - name: simulated-mount
    mountPath: /mnt/test
```

### Decision: Manual Failure Simulation via kubectl exec
**Rationale**: Per spec clarification - keeps tooling minimal, gives developers direct control.

**Commands documented in quickstart.md**:
```bash
# Simulate failure
kubectl exec -it <pod> -c main-app -- rm /mnt/test/.health-check

# Restore health
kubectl exec -it <pod> -c main-app -- sh -c 'echo "healthy" > /mnt/test/.health-check'
```

---

## 4. Probe Configuration

### Decision: HTTP Probes on Existing Health Endpoints
**Rationale**: Reuses monitor's existing `/healthz/live` and `/healthz/ready` endpoints. No new code required.

**Configuration**:
```yaml
livenessProbe:
  httpGet:
    path: /healthz/live
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /healthz/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 3
```

**Timing Rationale**:
- `periodSeconds: 5` for readiness - fast detection of mount failures
- `failureThreshold: 3` - aligns with monitor's debounce threshold (3 consecutive failures)
- Total time to unhealthy: 15s (3 failures × 5s period)

---

## 5. Cross-Platform Support

### Decision: Native Architecture Build
**Rationale**: KIND automatically uses appropriate node image for host architecture. Go cross-compilation handles binary. No QEMU emulation needed.

**Verification**:
```bash
# Makefile detects architecture
GOARCH ?= $(shell go env GOARCH)
```

**Alternatives Considered**:
- Force AMD64 with QEMU: Rejected - slow, unreliable on Apple Silicon
- Separate ARM64/AMD64 manifests: Rejected - unnecessary complexity

---

## 6. Makefile Target Design

### Decision: Compound Targets for Common Workflows
**Rationale**: Developers think in workflows, not individual commands.

| Target | Action | Dependencies |
|--------|--------|--------------|
| `kind-create` | Create cluster | Docker running |
| `kind-delete` | Delete cluster | None |
| `kind-load` | Build image, load to cluster | `docker` target |
| `kind-deploy` | Apply manifests | Cluster exists |
| `kind-redeploy` | Rebuild, reload, restart pods | `kind-load` |
| `kind-logs` | Tail monitor logs | Deployment exists |
| `kind-status` | Show pod/container status | Deployment exists |

### Decision: Idempotent Operations
**Rationale**: Running `make kind-create` twice should not error. Use `kind get clusters | grep` checks.

---

## 7. Test Application Container

### Decision: Alpine-based Sleep Container
**Rationale**: Minimal footprint, includes shell for debugging, `sleep infinity` keeps pod running.

```yaml
- name: main-app
  image: alpine:3.19
  command: ['sleep', 'infinity']
```

**Alternatives Considered**:
- busybox: Slightly smaller but alpine more debuggable
- nginx: Unnecessary HTTP server, heavier
- Custom app: Scope creep - testing the monitor, not the app

---

## 8. Documentation Structure

### Decision: Dedicated README in deploy/kind/
**Rationale**: Self-contained documentation close to manifests. Main README stays focused on project overview.

**Structure**:
1. Prerequisites (Docker, kubectl, KIND)
2. Quick Start (3-command workflow)
3. Failure Simulation Guide
4. Troubleshooting
5. Cleanup

---

## Sources

- [KIND Quick Start](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [Kubernetes Sidecar Containers](https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/)
- [Configure Liveness, Readiness Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [Kubernetes Image Pull Policy](https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy)
- [KIND Local Image Loading](https://kind.sigs.k8s.io/docs/user/quick-start/#loading-an-image-into-your-cluster)
