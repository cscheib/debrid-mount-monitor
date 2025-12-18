# Implementation Plan: Init Container Mode

**Branch**: `010-init-container-mode` | **Date**: 2025-12-17 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/010-init-container-mode/spec.md`

## Summary

Add a `--init-container-mode` flag that runs the service in a one-shot mode: load configuration, check all configured mounts once, log results, and exit with code 0 (all healthy) or 1 (any failure). This enables Kubernetes initContainer patterns to gate pod startup on mount availability.

## Technical Context

**Language/Version**: Go 1.21+ (required for log/slog structured logging)
**Primary Dependencies**: Standard library only + existing approved dependencies (pflag, go-multierror)
**Storage**: N/A (read-only canary file checks)
**Testing**: Go standard testing + matryer/is assertions
**Target Platform**: Linux containers (scratch/distroless), cross-compiled for arm64/amd64
**Project Type**: Single binary CLI application
**Performance Goals**: Complete all mount checks and exit within 30 seconds
**Constraints**: No new external dependencies; reuse existing health.Checker and config packages
**Scale/Scope**: Typically 1-5 mounts per configuration

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Minimal Dependencies | ✅ PASS | No new dependencies; reuses existing approved libs |
| II. Single Static Binary | ✅ PASS | Same binary, new mode via flag |
| III. Cross-Platform Compilation | ✅ PASS | No architecture-specific code |
| IV. Signal Handling | ✅ PASS | Init mode exits immediately; no long-running signal handling needed |
| V. Container Sidecar Design | ✅ PASS | Enables initContainer pattern; fast startup, stdout logging |
| VI. Fail-Safe Orchestration | ✅ PASS | Core purpose: gate startup on mount health |

**Gate Result**: ✅ All gates pass. No violations require justification.

## Project Structure

### Documentation (this feature)

```text
specs/010-init-container-mode/
├── plan.md              # This file
├── research.md          # Phase 0 output (minimal - no unknowns)
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/mount-monitor/
└── main.go              # Add --init-container-mode flag and runInitMode() function

internal/
├── config/
│   └── config.go        # Add InitContainerMode field to Config struct
└── health/
    ├── checker.go       # Existing - reused unchanged
    └── state.go         # Existing - reused unchanged

# Tests
internal/config/
└── config_test.go       # Add tests for new flag parsing
```

**Structure Decision**: Single project layout. This feature adds a new execution path to the existing binary without changing the package structure.

## Complexity Tracking

> No violations - table not needed.

---

## Phase 0: Research

### Research Summary

No unknowns requiring external research. The implementation approach is clear:

1. **Flag Parsing**: pflag (already approved) supports boolean flags trivially
2. **Health Checking**: Existing `health.Checker.Check()` performs exactly the needed canary file validation
3. **Exit Codes**: Go `os.Exit()` is the standard approach
4. **Logging**: Existing slog setup can be reused

**Decision Log**:

| Topic | Decision | Rationale |
|-------|----------|-----------|
| Check strategy | Sequential | Matches existing monitor behavior; simpler; predictable timeout behavior |
| Exit code | 0=success, 1=failure | Standard Unix convention; no benefit to differentiated error codes |
| Validation relaxation | Skip some validations | In init mode, HTTPPort, CheckInterval, ShutdownTimeout are irrelevant |

---

## Phase 1: Design

### Architecture Overview

```text
┌─────────────────────────────────────────────────────────────────┐
│                         main.go                                  │
├─────────────────────────────────────────────────────────────────┤
│  main()                                                          │
│    ├── config.Load() ─── adds --init-container-mode flag        │
│    ├── setupLogger()                                             │
│    │                                                             │
│    └── if cfg.InitContainerMode:                                │
│          └── runInitMode(cfg, logger) ──→ os.Exit(0 or 1)       │
│        else:                                                     │
│          └── [existing service startup...]                       │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    runInitMode(cfg, logger)                      │
├─────────────────────────────────────────────────────────────────┤
│  1. Create Mount objects from cfg.Mounts                         │
│  2. Create health.Checker with cfg.ReadTimeout                   │
│  3. For each mount (sequential):                                 │
│       result := checker.Check(ctx, mount)                        │
│       Log result (success or failure with error)                 │
│  4. If any failures: return exit code 1                          │
│     If all success: return exit code 0                           │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Input**: Config file (JSON) + `--init-container-mode` flag
2. **Processing**: Sequential canary file reads with timeout
3. **Output**: Structured logs (JSON/text) + exit code

### Key Implementation Details

**1. Config Changes** (`internal/config/config.go`):
- Add `InitContainerMode bool` field to `Config` struct
- Add `--init-container-mode` flag parsing in `Load()`
- Conditionally skip irrelevant validations in init mode (HTTPPort, CheckInterval, etc.)

**2. Main Entry Point** (`cmd/mount-monitor/main.go`):
- After config load and logger setup, check `cfg.InitContainerMode`
- If true, call `runInitMode()` and exit with returned code
- Never start HTTP server, monitor, or watchdog in init mode

**3. Init Mode Function** (new in `main.go`):
```go
func runInitMode(cfg *config.Config, logger *slog.Logger) int {
    // Create mounts
    mounts := make([]*health.Mount, len(cfg.Mounts))
    for i, mc := range cfg.Mounts {
        mounts[i] = health.NewMount(mc.Name, mc.Path, mc.CanaryFile, mc.FailureThreshold)
    }

    // Create checker
    checker := health.NewChecker(cfg.ReadTimeout)

    // Check all mounts
    ctx := context.Background()
    allHealthy := true

    for _, mount := range mounts {
        result := checker.Check(ctx, mount)
        if result.Success {
            logger.Info("mount check passed",
                "name", mount.Name,
                "path", mount.Path,
                "duration", result.Duration.String())
        } else {
            allHealthy = false
            logger.Error("mount check failed",
                "name", mount.Name,
                "path", mount.Path,
                "error", result.Error.Error(),
                "duration", result.Duration.String())
        }
    }

    if allHealthy {
        logger.Info("all mount checks passed", "count", len(mounts))
        return 0
    }
    logger.Error("one or more mount checks failed", "count", len(mounts))
    return 1
}
```

### Edge Cases Handled

| Case | Behavior |
|------|----------|
| No mounts configured | Config validation fails (existing behavior) |
| Config file missing | Exit with error before init mode runs |
| Mount path doesn't exist | Check fails with clear error |
| Canary file missing | Check fails with "file not found" |
| Canary read timeout | Check fails with context deadline exceeded |
| All mounts healthy | Log success, exit 0 |
| Any mount unhealthy | Log all results, exit 1 |

### Validation Adjustments

In init-container mode, these validations become irrelevant and should be skipped:
- `HTTPPort` - no HTTP server started
- `CheckInterval` - only one check performed
- `ShutdownTimeout` - immediate exit
- `Watchdog.*` - watchdog not started

Validations that still apply:
- At least one mount required
- Mount paths required
- `ReadTimeout` must be valid (used for canary checks)
- Log level/format must be valid

---

## Files to Modify

| File | Changes |
|------|---------|
| `internal/config/config.go` | Add `InitContainerMode` field; add flag parsing; adjust validation |
| `cmd/mount-monitor/main.go` | Add early exit path for init mode; add `runInitMode()` function |

## Files to Add

| File | Purpose |
|------|---------|
| `specs/010-init-container-mode/quickstart.md` | Usage documentation |

## Test Plan

| Test | Description |
|------|-------------|
| `TestLoad_InitContainerModeFlag` | Verify flag sets `InitContainerMode=true` |
| `TestValidate_InitContainerMode_SkipsIrrelevant` | Verify HTTPPort validation skipped in init mode |
| Integration test (manual) | Run with healthy/unhealthy mounts; verify exit codes |
