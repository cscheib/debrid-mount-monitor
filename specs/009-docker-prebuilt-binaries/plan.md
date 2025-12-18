# Implementation Plan: Docker Images Use Pre-built Binaries

**Branch**: `009-docker-prebuilt-binaries` | **Date**: 2025-12-17 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/009-docker-prebuilt-binaries/spec.md`

## Summary

Modify the release workflow's Docker job to use pre-compiled binary artifacts from the build job instead of recompiling Go source code. This eliminates redundant compilation, ensures version embedding in Docker images, and reduces CI time.

## Technical Context

**Language/Version**: YAML (GitHub Actions), Dockerfile
**Primary Dependencies**: docker buildx, actions/download-artifact@v6, actions/checkout@v6 (sparse)
**Storage**: N/A (CI/CD workflow only)
**Testing**: Manual workflow test with test tag
**Target Platform**: GitHub Actions ubuntu-latest runner
**Project Type**: CI/CD infrastructure
**Performance Goals**: Faster Docker build (eliminate Go compilation)
**Constraints**: Must maintain multi-arch support, backward compatibility with local dev
**Scale/Scope**: Single workflow file change + new Dockerfile

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Minimal Dependencies | ✅ PASS | No new runtime dependencies |
| II. Single Static Binary | ✅ PASS | Binaries unchanged, still static |
| III. Cross-Platform Compilation | ✅ PASS | Both ARM64 and AMD64 supported |
| IV. Signal Handling | ✅ N/A | CI/CD change, not runtime |
| V. Container Sidecar Design | ✅ PASS | Scratch base maintained |
| VI. Fail-Safe Orchestration | ✅ N/A | CI/CD change, not runtime |

**Gate Result**: PASS - No violations

## Project Structure

### Documentation (this feature)

```text
specs/009-docker-prebuilt-binaries/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0: Research findings
├── data-model.md        # Phase 1: Data flow model
├── quickstart.md        # Phase 1: Implementation guide
├── contracts/           # Phase 1: Artifact contracts
│   ├── Dockerfile.release
│   └── release-workflow-docker-job.yaml
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
# Files to be created/modified
Dockerfile.release       # NEW: Release-specific Dockerfile (copies pre-built binary)
.github/workflows/
└── release.yml          # MODIFIED: Docker job uses pre-built binaries

# Files unchanged
Dockerfile               # Unchanged: Local dev still compiles from source
Makefile                 # Unchanged: make docker uses Dockerfile
```

**Structure Decision**: Minimal change - one new file (Dockerfile.release) and one modified workflow. Existing Dockerfile preserved for local development backward compatibility.

## Implementation Approach

### Key Insight

Docker buildx automatically provides `TARGETARCH` build argument when using `--platform`. The Dockerfile can use `${TARGETARCH}` to select the correct pre-built binary without any conditional logic.

```dockerfile
COPY mount-monitor-linux-${TARGETARCH} /mount-monitor
```

### Changes Required

#### 1. New File: `Dockerfile.release`

Simple scratch-based Dockerfile that copies pre-built binary:
- Uses `ARG TARGETARCH` (auto-populated by buildx)
- COPY interpolates architecture into filename
- Same USER, EXPOSE, ENTRYPOINT as production Dockerfile

#### 2. Modified: `.github/workflows/release.yml` (Job 4)

Replace current Docker job with:
1. **Download artifacts**: Get binaries from build job
2. **Sparse checkout**: Only need Dockerfile.release
3. **Build context**: Directory with binaries + Dockerfile
4. **Buildx command**: Use Dockerfile.release, build from context

### What Stays the Same

- Build job (Job 2): Unchanged - still compiles binaries
- Verify job (Job 3): Unchanged - still validates binaries
- Release job (Job 5): Unchanged - still creates GitHub Release
- Dockerfile: Unchanged - local dev still works
- Makefile: Unchanged - `make docker` still works

## Complexity Tracking

> No constitution violations to justify.

| Aspect | Complexity | Justification |
|--------|-----------|---------------|
| New Dockerfile | Low | 8 lines, single purpose |
| Workflow changes | Low | Replace ~20 lines in one job |
| Testing | Low | Manual test with tag push |

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Artifact naming mismatch | Low | High | Use exact names from build job |
| TARGETARCH not interpolating | Low | High | Test with both platforms |
| Context path errors | Medium | Medium | Clear documentation |

## Success Verification

After implementation, verify:

1. **No Go compilation in Docker job**
   - Check workflow logs for absence of `go build`

2. **Version displayed correctly**
   ```bash
   docker pull ghcr.io/cscheib/debrid-mount-monitor:v1.0.0
   docker run ghcr.io/cscheib/debrid-mount-monitor:v1.0.0 --help
   # Should show: Version: 1.0.0
   ```

3. **Both architectures work**
   ```bash
   docker manifest inspect ghcr.io/cscheib/debrid-mount-monitor:v1.0.0
   # Should show linux/amd64 and linux/arm64
   ```

4. **Local dev unchanged**
   ```bash
   make docker
   # Should still work, compiling from source
   ```
