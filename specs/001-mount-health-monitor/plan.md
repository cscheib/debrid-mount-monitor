# Implementation Plan: Mount Health Monitor

**Branch**: `001-mount-health-monitor` | **Date**: 2025-12-14 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-mount-health-monitor/spec.md`

## Summary

A lightweight sidecar service that monitors debrid WebDAV mount health by reading canary files, exposing Kubernetes liveness/readiness probe endpoints. When mounts fail health checks (past debounce threshold), the liveness probe returns HTTP 503, triggering Kubernetes to restart the pod. Built as a single static Go binary with zero external dependencies.

## Technical Context

**Language/Version**: Go 1.21+ (required for log/slog structured logging)
**Primary Dependencies**: Standard library only (net/http, os/signal, context, log/slog, encoding/json, time, sync)
**Storage**: N/A (stateless - all state in memory, lost on restart by design)
**Testing**: go test (standard library testing package)
**Target Platform**: Linux containers (ARM64 + AMD64)
**Project Type**: Single project
**Performance Goals**: <100ms health endpoint response (SC-005), startup <5s (SC-006)
**Constraints**: <20MB container image, <10MB memory at runtime, 30s graceful shutdown timeout
**Scale/Scope**: Single monitor instance per pod (sidecar pattern), 1-10 mounts typical

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Implementation |
|-----------|--------|----------------|
| I. Minimal Dependencies | ✅ PASS | Standard library only; no third-party packages |
| II. Single Static Binary | ✅ PASS | `CGO_ENABLED=0 go build` produces static binary |
| III. Cross-Platform Compilation | ✅ PASS | `GOOS=linux GOARCH=arm64/amd64` built-in |
| IV. Signal Handling | ✅ PASS | `os/signal` package for SIGTERM/SIGINT |
| V. Container Sidecar Design | ✅ PASS | HTTP endpoints, stdout/stderr logging, no persistent storage |
| VI. Fail-Safe Orchestration | ✅ PASS | Debounce logic, liveness/readiness separation, logging |

**No violations requiring justification.**

## Project Structure

### Documentation (this feature)

```text
specs/001-mount-health-monitor/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (OpenAPI spec)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/
└── mount-monitor/
    └── main.go          # Entry point, signal handling, CLI flags

internal/
├── config/
│   └── config.go        # Configuration parsing (env vars, flags)
├── health/
│   ├── checker.go       # Mount health check logic (canary file read)
│   └── state.go         # Health state tracking, debounce logic
├── server/
│   └── server.go        # HTTP server for probe endpoints
└── monitor/
    └── monitor.go       # Main monitoring loop, orchestration

tests/
└── unit/
    ├── checker_test.go   # Health check unit tests
    ├── config_test.go    # Configuration parsing tests
    ├── monitor_test.go   # Monitor integration tests
    ├── server_test.go    # HTTP endpoint tests
    ├── shutdown_test.go  # Graceful shutdown tests
    └── state_test.go     # State management unit tests

build/
├── Dockerfile           # Multi-stage build, scratch base
└── Dockerfile.debug     # Alpine-based for debugging

.github/
└── workflows/
    └── ci.yml           # Build + test for ARM64/AMD64
```

**Structure Decision**: Single project layout with `cmd/` for entry point and `internal/` for private packages. This follows Go best practices for a small, focused application. No web frontend or mobile components.

## Complexity Tracking

> **No constitution violations to justify.**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| (none)    | N/A        | N/A                                 |
