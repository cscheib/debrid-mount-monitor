# Research: Tech Debt Cleanup

**Feature**: 007-tech-debt-cleanup
**Date**: 2025-12-16

## Overview

This document captures research decisions for the tech debt cleanup feature. Since this is a refactoring feature working with existing patterns, research focuses on best practices for safe refactoring and breaking change communication.

## Research Topics

### 1. Go Module Namespace Rename

**Decision**: Use `go mod edit -module` to rename, then update all imports via `gofmt -w -r` or sed.

**Rationale**:
- `go mod edit -module github.com/cscheib/debrid-mount-monitor` updates go.mod atomically
- Import path updates can be done with find/sed since imports are simple string replacements
- No external consumers exist (private project), so no module retraction needed

**Alternatives Considered**:
- Manual find/replace: Rejected - error prone for partial matches
- Using `gomvpkg`: Rejected - overkill for simple namespace change, adds tooling dependency

**Implementation Notes**:
- Order of operations: Update go.mod first, then imports, then verify with `go build`
- All imports follow pattern `github.com/chris/debrid-mount-monitor/internal/*`
- Replace `github.com/chris/` with `github.com/cscheib/` in all `.go` files

### 2. Terminology Alignment (debounce → failureThreshold)

**Decision**: Global rename from `debounce`/`Debounce` to `failure`/`Failure` terminology, aligning with existing per-mount `failureThreshold` field.

**Rationale**:
- Per-mount config already uses `failureThreshold` - consistency is paramount
- "Failure threshold" is self-documenting; "debounce" requires domain knowledge
- Clean break is preferable to maintaining dual terminology

**Alternatives Considered**:
- Keep both terms: Rejected - increases cognitive load, documentation confusion
- Use different term (e.g., `unhealthyThreshold`): Rejected - per-mount already uses `failureThreshold`

**Implementation Notes**:
- Config struct: `DebounceThreshold` → `FailureThreshold`
- JSON key: `debounceThreshold` → `failureThreshold`
- CLI flag: `--debounce-threshold` → `--failure-threshold`
- Variable names: `debounceThreshold` → `failureThreshold`
- Comments: Replace "debounce" terminology throughout
- Log messages: Update any threshold-related logging

**Breaking Change**: JSON config key change requires users to update config files. Document in CHANGELOG/release notes.

### 3. Path Configuration Documentation

**Decision**: Add explicit documentation stating mount paths can be relative or absolute, canary files are always relative to mount path.

**Rationale**:
- Current code handles both (uses `filepath.Join` which works with either)
- Users should not have to read source code to understand config options
- Prevents support requests and misconfiguration

**Alternatives Considered**:
- Require absolute paths only: Rejected - breaks flexibility for container deployments
- Auto-resolve relative to config file location: Rejected - adds complexity, current behavior (CWD-relative) is standard

**Implementation Notes**:
- Update README.md Configuration section
- Update JSON example to show both relative and absolute paths
- Clarify canary file is always relative to mount path (joined with `filepath.Join`)

### 4. Environment Variable Removal

**Decision**: Remove all configuration-related `os.Getenv` calls from `internal/config/config.go`. Preserve Kubernetes runtime detection variables.

**Rationale**:
- JSON config is more expressive (per-mount arrays, structured data)
- Reduces configuration precedence complexity (was: defaults → config → env → flags)
- New precedence is simpler: defaults → config → flags
- CLI flags remain for development/testing convenience

**Alternatives Considered**:
- Deprecation warning then removal: Rejected - overcomplicates for a cleanup feature
- Keep subset of env vars: Rejected - partial removal creates inconsistent UX

**Implementation Notes**:
- Remove `os.Getenv` blocks in `config.go` lines ~112-154
- Preserve `watchdog/k8s_client.go` env vars (`KUBERNETES_SERVICE_HOST`, `KUBERNETES_SERVICE_PORT`)
- Preserve `main.go` env vars (`POD_NAME`, `POD_NAMESPACE`) - these are runtime detection
- Remove environment variable tests in `config_file_test.go`
- Update README.md to remove environment variable documentation
- Document breaking change prominently in release notes

**Preserved Environment Variables** (not user configuration):
| Variable | Location | Purpose |
|----------|----------|---------|
| `KUBERNETES_SERVICE_HOST` | k8s_client.go | K8s API server detection |
| `KUBERNETES_SERVICE_PORT` | k8s_client.go | K8s API server detection |
| `POD_NAME` | main.go | Pod identity for watchdog |
| `POD_NAMESPACE` | main.go | Namespace for K8s API calls |

## Summary

All research items resolved. No NEEDS CLARIFICATION items remain.

| Topic | Decision | Breaking Change |
|-------|----------|-----------------|
| Namespace rename | `chris` → `cscheib` | No (internal refactor) |
| Terminology | `debounce` → `failureThreshold` | Yes (JSON config key) |
| Path docs | Document relative/absolute support | No (documentation only) |
| Env var removal | Remove all config env vars | Yes (migration required) |
