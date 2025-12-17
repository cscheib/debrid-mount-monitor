# Implementation Plan: Tech Debt Cleanup

**Branch**: `007-tech-debt-cleanup` | **Date**: 2025-12-16 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/007-tech-debt-cleanup/spec.md`

## Summary

This tech debt cleanup addresses four items: (1) Go module namespace rename from `github.com/chris/debrid-mount-monitor` to `github.com/cscheib/debrid-mount-monitor` to match the actual GitHub repository, (2) terminology alignment from "debounce" to "failureThreshold" for consistency, (3) documentation clarification for mount path formats, and (4) removal of environment variable configuration support to simplify the configuration model.

**Breaking Change**: Environment variable configuration removal requires users to migrate to JSON config or CLI flags.

## Technical Context

**Language/Version**: Go 1.21+ (required for log/slog structured logging)
**Primary Dependencies**: Standard library only (no external dependencies per constitution)
**Storage**: N/A (config file is read-only input)
**Testing**: `go test` with unit tests in `tests/unit/`
**Target Platform**: Linux containers (AMD64/ARM64), macOS for development
**Project Type**: Single binary CLI application
**Performance Goals**: N/A (refactoring feature, no new performance requirements)
**Constraints**: Must maintain all existing functionality; must pass all tests after changes
**Scale/Scope**: ~18 Go files affected across `cmd/`, `internal/`, and `tests/unit/`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Minimal Dependencies | ✅ PASS | No new dependencies; removing env var parsing reduces stdlib usage |
| II. Single Static Binary | ✅ PASS | No change to build output |
| III. Cross-Platform Compilation | ✅ PASS | No architecture-specific changes |
| IV. Signal Handling | ✅ PASS | No change to signal handling |
| V. Container Sidecar Design | ✅ PASS | No change to runtime behavior |
| VI. Fail-Safe Orchestration | ✅ PASS | No change to health check or restart logic |

**Configuration Compliance Note**: The constitution states "Configuration MUST be injectable via environment variables (primary method for containers) or command-line flags". This feature removes environment variable support for user configuration, keeping only CLI flags and JSON config. This is compliant because:
1. Constitution uses "or" (disjunctive) - CLI flags alone satisfy the requirement
2. Constitution explicitly allows config files: "Configuration files MAY be supported"
3. Kubernetes runtime variables (`POD_NAME`, etc.) remain unchanged - these are environment detection, not user configuration

**GATE STATUS**: ✅ PASS - Proceed to Phase 0

## Project Structure

### Documentation (this feature)

```text
specs/007-tech-debt-cleanup/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (minimal for refactoring)
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (N/A - no API changes)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/
└── mount-monitor/
    └── main.go              # Entry point, imports to update

internal/
├── config/
│   ├── config.go            # Config struct, env vars to remove, terminology change
│   ├── file.go              # JSON config parsing, terminology change
│   └── testing.go           # Test helpers
├── health/
│   ├── checker.go           # Health check logic
│   └── state.go             # State management, terminology in comments
├── monitor/
│   └── monitor.go           # Monitor loop, imports and variable names
├── server/
│   └── server.go            # HTTP endpoints, imports and comments
└── watchdog/
    ├── watchdog.go          # Pod restart logic
    └── k8s_client.go        # K8s API client (env vars preserved here)

tests/
└── unit/
    ├── checker_test.go      # Import update
    ├── config_file_test.go  # Import update, env var tests to remove
    ├── config_test.go       # Import update, terminology tests
    ├── monitor_test.go      # Import update
    ├── server_test.go       # Import update, terminology in tests
    ├── shutdown_test.go     # Import update
    ├── state_test.go        # Import update, terminology in tests
    └── watchdog_test.go     # Import update
```

**Structure Decision**: Existing single-project structure preserved. All changes are in-place modifications to existing files.

## Complexity Tracking

No violations requiring justification. All changes comply with constitution principles.

## Post-Design Constitution Re-Check

*Re-evaluated after Phase 1 design completion.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Minimal Dependencies | ✅ PASS | No dependencies added; `os.Getenv` removals reduce stdlib usage |
| II. Single Static Binary | ✅ PASS | No impact on build output |
| III. Cross-Platform Compilation | ✅ PASS | Pure refactoring, no platform-specific code |
| IV. Signal Handling | ✅ PASS | No changes to signal handling code |
| V. Container Sidecar Design | ✅ PASS | No runtime behavior changes |
| VI. Fail-Safe Orchestration | ✅ PASS | Terminology change only; fail-safe logic unchanged |

**FINAL GATE STATUS**: ✅ PASS - Ready for task generation

## Generated Artifacts

| Artifact | Path | Description |
|----------|------|-------------|
| Implementation Plan | `specs/007-tech-debt-cleanup/plan.md` | This file |
| Research | `specs/007-tech-debt-cleanup/research.md` | Decision rationale for all changes |
| Data Model | `specs/007-tech-debt-cleanup/data-model.md` | Config structure changes |
| Quickstart | `specs/007-tech-debt-cleanup/quickstart.md` | Implementation guide |
| Contracts | N/A | No API changes in this feature |
