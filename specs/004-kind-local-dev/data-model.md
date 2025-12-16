# Data Model: KIND Local Development Environment

**Feature**: 004-kind-local-dev | **Date**: 2025-12-15

## Overview

This feature involves infrastructure configuration rather than application data. The "data model" describes the Kubernetes resources and their relationships.

---

## Kubernetes Resources

### 1. KIND Cluster

**Resource Type**: KIND Configuration (not a K8s resource)

| Attribute | Type | Description |
|-----------|------|-------------|
| name | string | Cluster name (env: `KIND_CLUSTER_NAME`, default: `debrid-mount-monitor`) |
| nodes | array | Node definitions (single control-plane node) |
| image | string | Kubernetes node image (`kindest/node:v1.28.0`) |

**Lifecycle**:
- Created: `make kind-create`
- Deleted: `make kind-delete`
- State: Running in Docker containers

---

### 2. Namespace

**Resource Type**: `v1/Namespace`

| Attribute | Value | Description |
|-----------|-------|-------------|
| name | `mount-monitor-dev` | Isolated namespace for testing |

**Relationships**:
- Contains: ConfigMap, Deployment

---

### 3. ConfigMap

**Resource Type**: `v1/ConfigMap`

| Attribute | Value | Description |
|-----------|-------|-------------|
| name | `mount-monitor-config` | Monitor configuration |
| namespace | `mount-monitor-dev` | Owning namespace |

**Data Fields**:

| Key | Value | Description |
|-----|-------|-------------|
| `MOUNT_PATHS` | `/mnt/test` | Paths to monitor |
| `CANARY_FILE` | `.health-check` | Canary filename |
| `CHECK_INTERVAL` | `10s` | Faster interval for dev |
| `DEBOUNCE_THRESHOLD` | `3` | Failures before unhealthy |
| `LOG_LEVEL` | `debug` | Verbose logging for dev |
| `LOG_FORMAT` | `json` | Structured output |
| `HTTP_PORT` | `8080` | Health endpoint port |

---

### 4. Deployment

**Resource Type**: `apps/v1/Deployment`

| Attribute | Value | Description |
|-----------|-------|-------------|
| name | `test-app-with-monitor` | Deployment name |
| namespace | `mount-monitor-dev` | Owning namespace |
| replicas | `1` | Single replica for dev |

**Pod Template Structure**:

```
Pod
├── initContainers
│   └── init-canary (busybox:1.36)
│       └── Creates /mnt/test/.health-check
│
├── containers
│   ├── main-app (alpine:3.19)
│   │   └── Mounts: simulated-mount → /mnt/test
│   │
│   └── mount-monitor (mount-monitor:dev)
│       ├── Mounts: simulated-mount → /mnt/test (readOnly)
│       ├── EnvFrom: mount-monitor-config
│       ├── Probes: liveness, readiness
│       └── Port: 8080
│
└── volumes
    └── simulated-mount (emptyDir)
```

---

### 5. Volume: simulated-mount

**Resource Type**: `emptyDir` volume

| Attribute | Value | Description |
|-----------|-------|-------------|
| name | `simulated-mount` | Volume identifier |
| medium | `""` (default) | Backed by node filesystem |

**Mount Points**:

| Container | Path | Mode |
|-----------|------|------|
| init-canary | `/mnt/test` | ReadWrite |
| main-app | `/mnt/test` | ReadWrite |
| mount-monitor | `/mnt/test` | ReadOnly |

**State Transitions**:

```
[Pod Created]
     │
     ▼
┌─────────────────────┐
│ emptyDir created    │
│ (empty directory)   │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ init-canary runs    │
│ creates .health-check│
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│ Mount HEALTHY       │
│ (canary exists)     │
└─────────┬───────────┘
          │
          │ Developer removes canary via kubectl exec
          ▼
┌─────────────────────┐
│ Mount UNHEALTHY     │
│ (canary missing)    │
└─────────┬───────────┘
          │
          │ Developer recreates canary
          ▼
┌─────────────────────┐
│ Mount HEALTHY       │
│ (canary restored)   │
└─────────────────────┘
```

---

## Resource Relationships

```
KIND Cluster
└── Namespace: mount-monitor-dev
    ├── ConfigMap: mount-monitor-config
    │   └── Referenced by: Deployment (envFrom)
    │
    └── Deployment: test-app-with-monitor
        └── Pod
            ├── Volume: simulated-mount (emptyDir)
            │   ├── Mounted by: init-canary
            │   ├── Mounted by: main-app
            │   └── Mounted by: mount-monitor
            │
            ├── Init Container: init-canary
            │
            └── Containers
                ├── main-app
                └── mount-monitor
                    └── Probes → /healthz/live, /healthz/ready
```

---

## Environment Variables

The mount-monitor container receives configuration via ConfigMap:

| Variable | Source | Purpose |
|----------|--------|---------|
| `MOUNT_PATHS` | ConfigMap | Paths to monitor |
| `CANARY_FILE` | ConfigMap | Health check filename |
| `CHECK_INTERVAL` | ConfigMap | Check frequency |
| `DEBOUNCE_THRESHOLD` | ConfigMap | Failure threshold |
| `LOG_LEVEL` | ConfigMap | Logging verbosity |
| `LOG_FORMAT` | ConfigMap | Log output format |
| `HTTP_PORT` | ConfigMap | Health server port |

---

## Labels and Selectors

**Standard Labels** (applied to all resources):

| Label | Value | Purpose |
|-------|-------|---------|
| `app.kubernetes.io/name` | `mount-monitor` | Application name |
| `app.kubernetes.io/component` | `dev-environment` | Component type |
| `app.kubernetes.io/part-of` | `debrid-mount-monitor` | Parent project |

**Selector** (Deployment → Pod):

```yaml
selector:
  matchLabels:
    app: test-app-with-monitor
```
