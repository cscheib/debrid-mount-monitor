# Research: Pod Restart Watchdog

**Feature**: 005-pod-restart-watchdog
**Date**: 2025-12-15
**Status**: Complete

## Overview

This document captures research findings for implementing Kubernetes pod self-deletion using Go standard library only (no client-go dependency), in alignment with Constitution Principle I (Minimal Dependencies).

---

## 1. In-Cluster Authentication

### Decision
Use standard Kubernetes service account token authentication via files mounted by the kubelet.

### Rationale
- No external dependencies required - just file reads and standard TLS
- Token is auto-mounted by Kubernetes at predictable paths
- CA certificate enables secure TLS verification to API server

### Implementation Pattern

**File locations** (standard Kubernetes paths):
```
/var/run/secrets/kubernetes.io/serviceaccount/
├── token          # Bearer token for API authentication
├── ca.crt         # CA certificate for TLS verification
└── namespace      # Current namespace
```

**API server URL**: Derived from `KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT` environment variables (auto-set by kubelet).

### Alternatives Considered
| Alternative | Rejected Because |
|-------------|------------------|
| client-go library | Violates Constitution Principle I (Minimal Dependencies), adds ~50MB to binary |
| External kubeconfig | Not available in-cluster, requires file mount |
| Skip TLS verification | Security risk, not production-ready |

---

## 2. Pod Deletion API

### Decision
Use REST DELETE request to `/api/v1/namespaces/{namespace}/pods/{name}` endpoint.

### Rationale
- Standard Kubernetes API, well-documented
- Simple HTTP DELETE with JSON body for options
- Graceful deletion with configurable grace period

### API Details

**Endpoint**: `DELETE /api/v1/namespaces/{namespace}/pods/{name}`

**Headers**:
```
Authorization: Bearer {token}
Content-Type: application/json
```

**Request Body** (optional):
```json
{
  "gracePeriodSeconds": 30
}
```

**Response Status Codes**:

| Code | Meaning | Action |
|------|---------|--------|
| 200 OK | Pod deleted immediately | Success |
| 202 Accepted | Deletion initiated | Success |
| 404 Not Found | Pod doesn't exist | Idempotent success |
| 409 Conflict | Pod already terminating | Idempotent success |
| 401 Unauthorized | Token invalid | Permanent failure |
| 403 Forbidden | RBAC denied | Permanent failure |
| 500/503 | Server error | **Transient - retry** |

### Alternatives Considered
| Alternative | Rejected Because |
|-------------|------------------|
| Pod eviction API | More complex, designed for node draining not self-deletion |
| Patch deletionTimestamp | Less explicit than DELETE, same RBAC requirements |

---

## 3. Kubernetes Events API

### Decision
Create Event resources via POST to `/api/v1/namespaces/{namespace}/events` to record watchdog restarts.

### Rationale
- Standard Kubernetes pattern for recording operational events
- Visible via `kubectl get events`
- Enables alerting and audit trails

### Event Structure

```json
{
  "apiVersion": "v1",
  "kind": "Event",
  "metadata": {
    "name": "mount-monitor.{unix-nano}",
    "namespace": "{namespace}"
  },
  "involvedObject": {
    "apiVersion": "v1",
    "kind": "Pod",
    "name": "{pod-name}",
    "namespace": "{namespace}"
  },
  "reason": "WatchdogRestart",
  "message": "Mount {path} unhealthy, triggering pod restart",
  "type": "Warning",
  "firstTimestamp": "{RFC3339}",
  "lastTimestamp": "{RFC3339}",
  "count": 1,
  "source": {
    "component": "mount-monitor-watchdog"
  }
}
```

### Alternatives Considered
| Alternative | Rejected Because |
|-------------|------------------|
| Logging only | Not visible in `kubectl get events`, harder to correlate |
| Custom CRD | Overkill for simple event recording, requires additional RBAC |

---

## 4. Downward API for Pod Identity

### Decision
Use environment variables injected via Downward API to get pod name and namespace.

### Rationale
- Simpler than volume mounts for two small values
- Follows existing config pattern (environment variables)
- Well-supported in deployment manifests

### Implementation

**Deployment manifest**:
```yaml
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

**Go code**:
```go
podName := os.Getenv("POD_NAME")
podNamespace := os.Getenv("POD_NAMESPACE")
```

### Alternatives Considered
| Alternative | Rejected Because |
|-------------|------------------|
| Volume mounts | More complex for just two values |
| Parse from hostname | Unreliable, hostname may be customized |

---

## 5. Error Handling & Retry Strategy

### Decision
Implement exponential backoff with max 3 retries for transient failures, then exit with non-zero code as fallback.

### Rationale
- Aligns with spec clarification: "Fall back to process exit to trigger container restart via liveness probe"
- Exponential backoff prevents API server overload
- 3 retries balances reliability with response time

### Retry Configuration

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Max Retries | 3 | Per spec FR-009 |
| Initial Delay | 100ms | Fast first retry |
| Max Delay | 10s | Cap to prevent long waits |
| Backoff Multiplier | 2.0 | Standard exponential growth |

### Error Classification

**Transient (retry)**:
- HTTP 500 Internal Server Error
- HTTP 503 Service Unavailable
- Network connection errors
- Context deadline exceeded

**Permanent (fail immediately)**:
- HTTP 401 Unauthorized
- HTTP 403 Forbidden

**Idempotent (treat as success)**:
- HTTP 404 Not Found (pod already deleted)
- HTTP 409 Conflict (pod already terminating)

### Fallback Behavior
After all retries exhausted:
```go
logger.Error("pod deletion failed after retries, exiting for container restart")
os.Exit(1) // Triggers liveness probe failure → container restart
```

---

## 6. RBAC Validation at Startup

### Decision
Use SelfSubjectAccessReview API to validate RBAC permissions at startup.

### Rationale
- Fail fast if permissions are missing
- Clear error message helps operators debug RBAC issues
- Prevents watchdog from being enabled without proper permissions

### API Details

**Endpoint**: `POST /apis/authorization.k8s.io/v1/selfsubjectaccessreviews`

**Request**:
```json
{
  "apiVersion": "authorization.k8s.io/v1",
  "kind": "SelfSubjectAccessReview",
  "spec": {
    "resourceAttributes": {
      "verb": "delete",
      "resource": "pods",
      "namespace": "{namespace}"
    }
  }
}
```

**Response**:
```json
{
  "status": {
    "allowed": true,
    "reason": "RBAC: allowed by RoleBinding..."
  }
}
```

---

## 7. RBAC Resources Required

### Decision
Create namespace-scoped Role and RoleBinding (not ClusterRole).

### Rationale
- Principle of least privilege: only access own namespace
- Simpler RBAC model for sidecar pattern
- No cluster-wide permissions needed

### Required Permissions

| Resource | Verbs | Purpose |
|----------|-------|---------|
| pods | delete, get | Delete own pod, check termination status |
| events | create | Record watchdog restart events |
| selfsubjectaccessreviews | create | Validate RBAC at startup |

### RBAC Manifest Structure

```yaml
# ServiceAccount
apiVersion: v1
kind: ServiceAccount
metadata:
  name: mount-monitor
  namespace: {namespace}

# Role (namespace-scoped)
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: mount-monitor-watchdog
  namespace: {namespace}
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["delete", "get"]
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create"]
- apiGroups: ["authorization.k8s.io"]
  resources: ["selfsubjectaccessreviews"]
  verbs: ["create"]

# RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: mount-monitor-watchdog
  namespace: {namespace}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: mount-monitor-watchdog
subjects:
- kind: ServiceAccount
  name: mount-monitor
  namespace: {namespace}
```

---

## 8. Out-of-Cluster Detection

### Decision
Detect out-of-cluster environment by checking for service account token file.

### Rationale
- Simple file existence check
- No false positives (file only exists in-cluster)
- Enables graceful degradation for local development

### Implementation

```go
func IsInCluster() bool {
    _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token")
    return err == nil
}
```

When running outside Kubernetes:
- Log warning that watchdog mode is disabled
- Continue operating in monitoring-only mode
- Return HTTP 503 on probes as before (no pod deletion)

---

## Summary

All technical unknowns have been resolved. The implementation will:

1. Use standard library `net/http` for all Kubernetes API calls
2. Read service account token from mounted files
3. Use environment variables for pod identity (Downward API)
4. Implement exponential backoff retry with process exit fallback
5. Validate RBAC permissions at startup
6. Create namespace-scoped RBAC resources
7. Gracefully degrade when running outside Kubernetes

No constitution violations. Proceed to Phase 1 design.
