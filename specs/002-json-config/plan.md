# Implementation Plan: JSON Configuration File

**Branch**: `002-json-config` | **Date**: 2025-12-14 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-json-config/spec.md`

## Summary

Add JSON configuration file support to the mount health monitor, enabling per-mount configuration (name, canary file, failure threshold) while preserving backwards compatibility with existing environment variable and CLI flag configuration. The configuration precedence becomes: Defaults → Config File → Env Vars → CLI Flags.

## Technical Context

**Language/Version**: Go 1.21+ (required for log/slog structured logging)
**Primary Dependencies**: Standard library only (encoding/json, os, flag, path/filepath)
**Storage**: N/A (configuration file is read-only input)
**Testing**: Go testing package with t.TempDir() for file isolation
**Target Platform**: Linux containers (ARM64 and AMD64), scratch/distroless base
**Project Type**: Single CLI application
**Performance Goals**: Configuration loading < 100ms, no impact on runtime performance
**Constraints**: No external dependencies, static binary, < 20MB container image
**Scale/Scope**: Support 10+ mounts per config file (per SC-001)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Minimal Dependencies | ✅ PASS | Uses only `encoding/json` from stdlib for JSON parsing |
| II. Single Static Binary | ✅ PASS | Config file is optional; binary still runs without it |
| III. Cross-Platform Compilation | ✅ PASS | No platform-specific code needed for JSON parsing |
| IV. Signal Handling | ✅ PASS | No changes to signal handling required |
| V. Container Sidecar Design | ✅ PASS | Config file can be mounted via ConfigMap; falls back to env vars |
| VI. Fail-Safe Orchestration | ✅ PASS | Invalid config fails startup (fail-fast behavior) |

**Constitution Note on Config Files**: The constitution states "Configuration files MAY be supported but MUST NOT be required." This feature complies by making the config file optional—the system falls back to env vars/CLI flags when no config file is present.

## Project Structure

### Documentation (this feature)

```text
specs/002-json-config/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (JSON schema)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/
└── mount-monitor/
    └── main.go              # Entry point - mount creation logic changes

internal/
├── config/
│   ├── config.go            # Add JSON file loading, per-mount config structs
│   └── file.go              # NEW: JSON file parsing and validation
├── health/
│   ├── state.go             # Add Name field to Mount struct
│   └── checker.go           # No changes needed
├── monitor/
│   └── monitor.go           # No changes needed
└── server/
    └── server.go            # Include mount name in status responses

tests/
└── unit/
    ├── config_test.go       # Add JSON config file tests
    ├── config_file_test.go  # NEW: Dedicated file parsing tests
    ├── state_test.go        # Add mount name tests
    └── server_test.go       # Add mount name in response tests
```

**Structure Decision**: Extends existing single-project structure. New `file.go` isolates JSON parsing logic from existing config.go to maintain separation of concerns and testability.

## Complexity Tracking

> No constitution violations to justify. All principles are satisfied.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| (none)    | N/A        | N/A                                 |

## Architecture Overview

### Configuration Loading Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                      config.Load()                               │
├─────────────────────────────────────────────────────────────────┤
│ 1. DefaultConfig()           → Base defaults                     │
│ 2. LoadFromFile()            → JSON file (if exists)      [NEW]  │
│    - Check --config flag                                         │
│    - Check ./config.json default                                 │
│    - Parse JSON, apply per-mount configs                         │
│ 3. applyEnvironmentVariables() → Env var overrides              │
│ 4. applyCommandLineFlags()   → CLI flag overrides               │
│ 5. Validate()                → Fail-fast on errors               │
└─────────────────────────────────────────────────────────────────┘
```

### Per-Mount Configuration Model

```
┌─────────────────────────────────────────────────────────────────┐
│ Config File (JSON)                                               │
├─────────────────────────────────────────────────────────────────┤
│ {                                                                │
│   "checkInterval": "30s",        // Global setting               │
│   "logLevel": "info",            // Global setting               │
│   "mounts": [                                                    │
│     {                                                            │
│       "name": "movies",          // Per-mount (optional)         │
│       "path": "/mnt/movies",     // Per-mount (required)         │
│       "canaryFile": ".ready",    // Per-mount (optional)         │
│       "failureThreshold": 5      // Per-mount (optional)         │
│     }                                                            │
│   ]                                                              │
│ }                                                                │
└─────────────────────────────────────────────────────────────────┘
```

## Implementation Phases

### Phase 1: Core JSON Loading (P1 - Load Configuration from JSON File)

**Goal**: Enable loading mount configurations from a JSON file

**Files Modified**:
- `internal/config/config.go` - Add ConfigFilePath field, update Load()
- `internal/config/file.go` - NEW: JSON parsing logic
- `tests/unit/config_file_test.go` - NEW: File parsing tests

**Key Changes**:
1. Add `--config` / `-c` CLI flag
2. Add default `./config.json` check logic
3. Parse JSON structure with global + per-mount settings
4. Integrate into existing Load() precedence chain

### Phase 2: Per-Mount Data Model (P2 - Per-Mount Configuration)

**Goal**: Support individual settings per mount

**Files Modified**:
- `internal/config/config.go` - Add MountConfig struct
- `internal/health/state.go` - Add Name field to Mount
- `cmd/mount-monitor/main.go` - Update mount creation loop
- `tests/unit/state_test.go` - Mount name tests

**Key Changes**:
1. Create `MountConfig` struct with per-mount fields
2. Add `Name` field to `health.Mount`
3. Update mount creation to use per-mount configs
4. Each mount gets its own canary file and threshold

### Phase 3: Validation & Logging (P3 - Configuration Validation)

**Goal**: Validate config and provide verbose startup logging

**Files Modified**:
- `internal/config/file.go` - Add validation functions
- `internal/config/config.go` - Update Validate()
- `internal/server/server.go` - Include mount name in status
- `cmd/mount-monitor/main.go` - Add verbose config logging

**Key Changes**:
1. Validate JSON structure and required fields
2. Validate per-mount settings (threshold >= 0 where 0 = use default, path required)
3. Log config source, all mount paths, and settings at info level
4. Include mount name in health status responses

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Breaking backwards compatibility | Low | High | Extensive testing of env var/CLI-only scenarios |
| Config file permission issues | Medium | Medium | Clear error messages with file path |
| JSON parsing edge cases | Low | Low | Use stdlib encoding/json with strict validation |

## Testing Strategy

1. **Unit Tests**: JSON parsing, validation, precedence scenarios
2. **Integration Tests**: End-to-end config loading with temp files
3. **Backwards Compatibility Tests**: Existing env var/CLI configurations still work
4. **Edge Case Tests**: Malformed JSON, missing fields, empty file, file not found
