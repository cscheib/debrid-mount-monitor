# Evaluation: Adding Kubernetes Client Libraries to debrid-mount-monitor

> **Date**: 2025-12-16
> **Status**: Decided - Keep raw HTTP implementation

## Executive Summary

**Recommendation: Do NOT add Kubernetes client libraries (client-go)**

The current raw HTTP implementation is well-suited to this project's needs and philosophy. Adding `client-go` would violate the project's constitutional principles without providing meaningful benefits for the current use case.

---

## Current State Analysis

### Kubernetes API Usage

The project interacts with Kubernetes in a **limited, well-defined scope**:

| Operation | API Endpoint | Complexity |
|-----------|--------------|------------|
| Delete Pod | `DELETE /api/v1/namespaces/{ns}/pods/{name}` | Simple REST call |
| Check Pod Status | `GET /api/v1/namespaces/{ns}/pods/{name}` | Simple REST call |
| Create Event | `POST /api/v1/namespaces/{ns}/events` | JSON payload construction |
| RBAC Check | `POST /apis/authorization.k8s.io/v1/selfsubjectaccessreviews` | JSON payload construction |

**Total implementation**: ~400 lines in `internal/watchdog/k8s_client.go`

### Current Implementation Quality

The existing code is **production-ready and well-designed**:

- ✅ Proper TLS certificate validation
- ✅ Connection pooling (10 max idle connections)
- ✅ 30-second timeout (matches client-go default)
- ✅ Response body size limits (1MB) to prevent memory exhaustion
- ✅ Idempotent operations (404/409 treated as success for deletions)
- ✅ Permanent vs transient error distinction for retry logic
- ✅ Exponential backoff retry strategy
- ✅ Graceful degradation (disables if not in cluster or RBAC fails)
- ✅ Context-based cancellation support

---

## client-go Analysis

### What client-go Would Provide

| Feature | Value for This Project |
|---------|------------------------|
| Type-safe API objects | Low - only using 4 simple operations |
| Automatic retry/backoff | Already implemented |
| Watch/informer patterns | Not needed - no list/watch operations |
| Multi-version API support | Not needed - using stable v1 APIs |
| Kubeconfig file support | Not needed - only in-cluster auth |
| Leader election | Not needed - sidecar pattern |
| Dynamic client | Not needed - static operations |
| Shared informers | Not needed - no caching required |

### Cost of Adding client-go

| Impact | Magnitude |
|--------|-----------|
| Binary size increase | **+50-70MB** (current: ~10MB, would become ~60-80MB) |
| Transitive dependencies | **~100+ packages** added to go.mod |
| Build time | **2-3x slower** due to dependency compilation |
| Container image size | **Exceeds 20MB target** (Constitution §Build & Distribution) |
| Attack surface | Significant increase in third-party code |
| Maintenance burden | Track k8s.io/client-go releases (frequent updates) |

### Dependency Tree Comparison

**Current (spf13/pflag only)**:
```
go.mod: 1 direct dependency, 0 transitive
```

**With client-go**:
```
go.mod: 1 direct dependency → 80+ transitive
Including: apimachinery, api, utils, klog, json-iterator,
gogo/protobuf, go-openapi/*, golang/protobuf, etc.
```

---

## Constitutional Analysis

### Principle I: Minimal Dependencies

> "Standard library solutions MUST be preferred over third-party packages"

**Verdict**: Current implementation satisfies this completely. Adding client-go would be a clear violation.

### Principle II: Single Static Binary

> "Image size SHOULD be minimized (target: < 20MB uncompressed)"

**Verdict**: client-go's ~50MB footprint would exceed the constitutional target by 3x+.

### Dependency Approval Criteria

From the constitution:
- ✅ MUST solve a problem stdlib cannot reasonably address → **FAILS** (already solved with stdlib)
- ✅ MUST have zero or minimal transitive dependencies → **FAILS** (100+ deps)
- ✅ MUST NOT increase binary size significantly (< 1MB) → **FAILS** (+50MB)
- ✅ MUST NOT introduce CGO requirements → **PASSES** (pure Go)

**Result**: client-go fails 3 of 4 approval criteria.

---

## Feature Gap Analysis

### What the Current Implementation Lacks

| Missing Feature | Impact | Mitigation |
|-----------------|--------|------------|
| Watch API support | None - not needed | Polling works for this use case |
| Token refresh | Low - in-cluster tokens outlive pods | Not needed for sidecar pattern |
| API version negotiation | None - using stable v1 | Kubernetes maintains v1 indefinitely |
| Structured error types | Low - custom types work well | Already implemented |
| List operations | None - not used | Not needed |

### Scenarios Where client-go Would Make Sense

The project would benefit from client-go **only if**:

1. **Watch-based patterns needed** (e.g., watching multiple pods/deployments)
2. **Complex CRD operations** (creating/managing custom resources)
3. **Multi-cluster support** (external kubeconfig files)
4. **Operator pattern** (controller-runtime framework)
5. **Shared informer caching** (list-and-watch for many resources)

**None of these apply** to the debrid-mount-monitor use case.

---

## Alternative Considerations

### Lightweight Alternatives

If the raw HTTP approach ever becomes problematic, consider these before client-go:

| Library | Size Impact | Notes |
|---------|-------------|-------|
| Keep current | 0 | Best for current needs |
| `k8s.io/client-go/rest` only | ~5-10MB | Just the REST client, no typed clients |
| Custom codegen for needed types | ~1-2MB | Generate only pod/event types |

**Recommendation**: Keep current implementation unless scope significantly expands.

---

## When to Revisit This Decision

Reconsider adding client-go if the project needs to:

- [ ] List/watch multiple pods across namespaces
- [ ] Manage Kubernetes resources beyond pod deletion
- [ ] Implement controller/operator patterns
- [ ] Support CRDs or custom API resources
- [ ] Run outside Kubernetes clusters with kubeconfig files
- [ ] Implement leader election for HA

---

## Conclusion

| Factor | Current (raw HTTP) | With client-go |
|--------|-------------------|----------------|
| Constitution compliance | ✅ Full | ❌ Violates 3 principles |
| Binary size | ✅ ~10MB | ❌ ~60-80MB |
| Dependencies | ✅ 1 | ❌ 100+ |
| Feature coverage | ✅ 100% of needs | ✅ 100% + unused features |
| Maintenance | ✅ Self-contained | ❌ Track upstream releases |
| Code complexity | ✅ 400 lines | ❌ External API changes |

**Final Verdict**: The ~400 lines of custom Kubernetes client code is an excellent trade-off. It provides exactly what the project needs without the bloat. This is a case where "reinventing the wheel" produces a better outcome than using a heavy external library.

---

## Files Referenced

- `internal/watchdog/k8s_client.go` - Current K8s client implementation (397 lines)
- `.specify/memory/constitution.md` - Project constitution (v1.2.0)
- `go.mod` - Current dependencies (only spf13/pflag)
