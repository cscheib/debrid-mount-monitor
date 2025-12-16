# Implementation Plan: Pod Restart Watchdog

**Branch**: `005-pod-restart-watchdog` | **Date**: 2025-12-15 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/005-pod-restart-watchdog/spec.md`

## Summary

Add watchdog mode to the mount health monitor that triggers **pod-level restarts** (not just container restarts) via the Kubernetes API when mounts become unhealthy. This ensures all containers in a pod restart together with fresh mount connections, preventing the main application from running with broken mounts.

**Technical Approach**: Extend the existing monitor with a Kubernetes API client that deletes the pod when mount state transitions to UNHEALTHY. Uses in-cluster authentication and requires RBAC resources for pod deletion permissions.

## Technical Context

**Language/Version**: Go 1.21+ (required for log/slog structured logging)
**Primary Dependencies**: Standard library only (net/http, encoding/json, os, time, context, log/slog) + Kubernetes REST API via net/http (no client-go dependency)
**Storage**: N/A (no persistent storage required)
**Testing**: Go standard testing package (`go test`)
**Target Platform**: Linux containers (Kubernetes pods)
**Project Type**: Single binary sidecar service
**Performance Goals**: Pod deletion triggered within 60 seconds of sustained mount failure
**Constraints**: <128MB memory, minimal CPU, must work in scratch/distroless containers
**Scale/Scope**: Single pod (self-deletion), 1-10 mount points per pod

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Minimal Dependencies | ✅ PASS | Using standard library net/http for K8s API calls, no client-go |
| II. Single Static Binary | ✅ PASS | No new runtime dependencies, remains scratch-compatible |
| III. Cross-Platform Compilation | ✅ PASS | Standard library only, no CGO required |
| IV. Signal Handling | ✅ PASS | Existing signal handling preserved, watchdog respects shutdown |
| V. Container Sidecar Design | ✅ PASS | Designed for sidecar pattern, minimal resource overhead |
| VI. Fail-Safe Orchestration | ✅ PASS | Core feature: triggers pod restart on mount failure |

**Gate Result**: PASS - No constitution violations. Proceed to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/005-pod-restart-watchdog/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (RBAC manifests)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/
└── mount-monitor/
    └── main.go              # Add watchdog initialization

internal/
├── config/
│   ├── config.go            # Add watchdog config fields
│   └── file.go              # Extend JSON config parsing
├── health/
│   ├── checker.go           # (unchanged)
│   └── state.go             # (unchanged)
├── monitor/
│   └── monitor.go           # Add watchdog trigger on UNHEALTHY
├── server/
│   └── server.go            # (unchanged)
└── watchdog/                # NEW PACKAGE
    ├── watchdog.go          # Watchdog state machine
    └── k8s_client.go        # Kubernetes API client (pod delete, event create)

deploy/
└── kind/
    ├── deployment.yaml      # Add ServiceAccount reference, env vars
    ├── rbac.yaml            # NEW: ServiceAccount, Role, RoleBinding
    └── configmap.yaml       # Add watchdog config fields

tests/
└── unit/
    └── watchdog_test.go     # NEW: Watchdog unit tests
```

**Structure Decision**: Extend existing single-project structure with new `internal/watchdog` package. Follows existing patterns in `internal/health` and `internal/monitor`.

## Complexity Tracking

> No constitution violations requiring justification.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| (none)    | -          | -                                   |
