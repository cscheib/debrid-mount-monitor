# Data Model: JSON Configuration File

**Feature**: 002-json-config
**Date**: 2025-12-14

## Overview

This document defines the data structures for JSON configuration file support, including new types and modifications to existing types.

---

## New Types

### FileConfig

Represents the JSON configuration file structure.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| checkInterval | string (duration) | No | "30s" | Time between health checks |
| readTimeout | string (duration) | No | "5s" | Timeout for canary file read |
| shutdownTimeout | string (duration) | No | "30s" | Graceful shutdown timeout |
| debounceThreshold | int | No | 3 | Consecutive failures before unhealthy |
| httpPort | int | No | 8080 | Port for health endpoints |
| logLevel | string | No | "info" | Log level (debug/info/warn/error) |
| logFormat | string | No | "json" | Log format (json/text) |
| canaryFile | string | No | ".health-check" | Default canary file for all mounts |
| mounts | []MountConfig | No | [] | Array of mount configurations |

**Validation Rules**:
- If `mounts` is provided and non-empty, mount paths from file take precedence
- Duration strings must be parseable by `time.ParseDuration()`
- If file specifies any value, it overrides defaults but not env vars/CLI flags

---

### MountConfig

Represents per-mount configuration within the JSON file.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| name | string | No | (derived from path) | Human-readable mount identifier |
| path | string | **Yes** | - | Filesystem path to mount point |
| canaryFile | string | No | (global default) | Canary file path relative to mount |
| failureThreshold | int | No | (global default) | Consecutive failures for this mount |

**Validation Rules**:
- `path` is required and must be non-empty
- `failureThreshold` must be >= 1 if specified
- `name` if not specified, logs will use path as identifier

---

## Modified Types

### Config (existing)

Add new field for config file path tracking.

| Field | Type | Change | Description |
|-------|------|--------|-------------|
| ConfigFile | string | **NEW** | Path to loaded config file ("" if none) |
| Mounts | []MountConfig | **NEW** | Replaces MountPaths for per-mount config |

**Migration Notes**:
- `MountPaths []string` becomes internal/derived from `Mounts`
- For backwards compatibility, env var `MOUNT_PATHS` still populates mounts (with global defaults)

---

### Mount (health package)

Add name field for identification.

| Field | Type | Change | Description |
|-------|------|--------|-------------|
| Name | string | **NEW** | Human-readable identifier (may be empty) |
| FailureThreshold | int | **NEW** | Per-mount threshold (previously global) |

**Constructor Change**:
```go
// Before
func NewMount(path, canaryFile string) *Mount

// After
func NewMount(name, path, canaryFile string, failureThreshold int) *Mount
```

---

## State Transitions

No new state transitions. Existing health states remain unchanged:
- `StatusUnknown` → `StatusHealthy` (on first successful check)
- `StatusHealthy` → `StatusDegraded` (on failure, count < threshold)
- `StatusDegraded` → `StatusUnhealthy` (on failure, count >= threshold)
- `StatusUnhealthy` → `StatusHealthy` (on successful check)
- `StatusDegraded` → `StatusHealthy` (on successful check before threshold)

---

## Relationships

```
┌─────────────────┐
│   FileConfig    │
│  (JSON input)   │
└────────┬────────┘
         │ parsed into
         ▼
┌─────────────────┐       ┌─────────────────┐
│     Config      │◄──────│  MountConfig[]  │
│   (runtime)     │       │  (per-mount)    │
└────────┬────────┘       └─────────────────┘
         │ creates
         ▼
┌─────────────────┐
│    Mount[]      │
│ (health state)  │
└─────────────────┘
```

---

## JSON Schema Reference

See `contracts/config-schema.json` for the formal JSON Schema definition.
