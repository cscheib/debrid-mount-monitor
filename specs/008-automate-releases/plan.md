# Implementation Plan: Automate Releases

**Branch**: `008-automate-releases` | **Date**: 2025-12-17 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/008-automate-releases/spec.md`

## Summary

Create a GitHub Actions release workflow that automatically builds and publishes binary artifacts and Docker images when a semantic version tag is pushed. The workflow will:
- Trigger on `v*` tags (including pre-releases like `v1.0.0-beta.1`)
- Build statically-linked binaries for linux/amd64 and linux/arm64
- Generate SHA256 checksums for all binaries
- Build and push multi-architecture Docker images to ghcr.io
- Create GitHub Releases with auto-generated release notes
- Support manual workflow dispatch for emergency releases

## Technical Context

**Language/Version**: YAML (GitHub Actions workflow), Go 1.21+ (existing project)
**Primary Dependencies**: GitHub Actions (actions/checkout@v6, actions/setup-go@v6, docker/build-push-action@v6, softprops/action-gh-release)
**Storage**: N/A (CI/CD workflow only)
**Testing**: Workflow validation via act (local) or test tag push to fork
**Target Platform**: GitHub Actions runners (ubuntu-latest)
**Project Type**: CI/CD infrastructure (single workflow file)
**Performance Goals**: Complete release within 15 minutes of tag push
**Constraints**: Must use existing Makefile/Dockerfile, retry transient failures (2 retries), no additional dependencies to main project
**Scale/Scope**: Single workflow file (~150-200 lines YAML)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Minimal Dependencies | ✅ PASS | No new Go dependencies; uses GitHub Actions ecosystem |
| II. Single Static Binary | ✅ PASS | Workflow produces statically-linked binaries (CGO_ENABLED=0) |
| III. Cross-Platform Compilation | ✅ PASS | Builds linux/amd64 and linux/arm64 as required |
| IV. Signal Handling | ✅ N/A | CI/CD workflow, not runtime behavior |
| V. Container Sidecar Design | ✅ PASS | Uses existing Dockerfile with scratch base |
| VI. Fail-Safe Orchestration | ✅ PASS | Release blocked on CI test failure (FR-012) |

**Build & Distribution Requirements**:
- ✅ Multi-architecture manifests via docker/build-push-action with `platforms: linux/amd64,linux/arm64`
- ✅ Minimal base image (existing Dockerfile uses `scratch`)
- ✅ Target image size < 20MB (existing constraint preserved)

**All gates passed.** Proceeding to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/008-automate-releases/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0: GitHub Actions best practices
├── data-model.md        # Phase 1: N/A (no data model for CI/CD)
├── quickstart.md        # Phase 1: How to create a release
└── tasks.md             # Phase 2: Implementation tasks
```

### Source Code (repository root)

```text
.github/
└── workflows/
    ├── ci.yml           # Existing CI workflow (reference)
    ├── integration.yml  # Existing integration tests (reference)
    └── release.yml      # NEW: Release workflow (this feature)

Makefile                 # Existing build targets (reuse)
Dockerfile               # Existing multi-stage build (reuse)
```

**Structure Decision**: CI/CD infrastructure feature - single new workflow file (`.github/workflows/release.yml`) that leverages existing Makefile targets and Dockerfile. No changes to application source code.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

*No violations - all constitution gates passed.*

---

## Post-Design Constitution Re-Check

*Performed after Phase 1 design completion.*

| Principle | Status | Design Validation |
|-----------|--------|-------------------|
| I. Minimal Dependencies | ✅ PASS | Workflow uses only GitHub Actions marketplace actions; no new Go deps |
| II. Single Static Binary | ✅ PASS | `CGO_ENABLED=0` in workflow, strip symbols with `-s -w` |
| III. Cross-Platform Compilation | ✅ PASS | Matrix build for linux/amd64 and linux/arm64 |
| V. Container Sidecar Design | ✅ PASS | Existing Dockerfile (scratch base) reused unchanged |
| VI. Fail-Safe Orchestration | ✅ PASS | `test` job gates all subsequent jobs via `needs: test` |

**All gates passed post-design.**

---

## Generated Artifacts

| Artifact | Path | Description |
|----------|------|-------------|
| Research | `specs/008-automate-releases/research.md` | GitHub Actions best practices research |
| Quickstart | `specs/008-automate-releases/quickstart.md` | How to create releases guide |
| Contract | `specs/008-automate-releases/contracts/release-workflow.yaml` | Release workflow design specification |

---

## Next Steps

Run `/speckit.tasks` to generate implementation tasks from this plan.
