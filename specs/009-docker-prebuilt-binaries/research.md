# Research: Docker Images with Pre-built Binaries

**Feature**: 009-docker-prebuilt-binaries
**Date**: 2025-12-17

## Research Questions

### 1. Can Docker buildx use pre-built binaries without recompiling?

**Decision**: Yes, using `ARG TARGETARCH` automatic build arguments

**Rationale**: Docker buildx automatically provides `TARGETARCH`, `TARGETOS`, and `TARGETPLATFORM` build arguments when using `--platform`. A Dockerfile can reference these to copy the correct pre-built binary.

**Alternatives Considered**:
- Manual manifest merge (more complex, requires separate image tags)
- Native runners per architecture (overkill when binaries already exist)

### 2. What Dockerfile pattern works for pre-built binaries?

**Decision**: Simple scratch-based Dockerfile with TARGETARCH interpolation

```dockerfile
FROM scratch
ARG TARGETARCH
COPY mount-monitor-linux-${TARGETARCH} /mount-monitor
USER 65534
ENTRYPOINT ["/mount-monitor"]
```

**Rationale**:
- Scratch base keeps image minimal (constitution requirement)
- TARGETARCH is auto-populated by buildx for each platform
- No Go toolchain needed in image

**Alternatives Considered**:
- Alpine base (larger, adds shell)
- Distroless (larger than scratch, adds nothing for static binary)

### 3. How to pass binaries to Docker build context?

**Decision**: Download artifacts to working directory, use as build context

**Rationale**:
- `actions/download-artifact@v6` can download to any path
- Binaries placed in context directory before `docker buildx build`
- Buildx sees them as regular files in context

**Pattern**:
```yaml
- name: Download amd64 binary
  uses: actions/download-artifact@v6
  with:
    name: mount-monitor-linux-amd64
    path: ./docker-context/

- name: Download arm64 binary
  uses: actions/download-artifact@v6
  with:
    name: mount-monitor-linux-arm64
    path: ./docker-context/
```

### 4. How does buildx handle multi-arch with pre-built binaries?

**Decision**: Single `docker buildx build --platform linux/amd64,linux/arm64` command

**Rationale**:
- Buildx builds each platform sequentially
- For each platform, TARGETARCH is set automatically
- The Dockerfile COPY uses the interpolated path
- Buildx creates manifest list automatically when pushing

**Key Insight**: QEMU is NOT needed when binaries are pre-built. Buildx only needs QEMU for RUN instructions that execute code. COPY instructions work without emulation.

### 5. Backward compatibility with local development?

**Decision**: Create separate `Dockerfile.release` for CI, keep existing `Dockerfile` for local dev

**Rationale**:
- Local developers use `make docker` which invokes existing Dockerfile
- Existing Dockerfile compiles from source (developers have Go installed)
- Release workflow uses `Dockerfile.release` which copies pre-built binaries
- No breaking change to local workflow

**Alternatives Considered**:
- Single Dockerfile with build args (complex, error-prone)
- Modify existing Dockerfile (breaks local workflow)

## Implementation Approach

### Workflow Changes (release.yml)

1. **Add download steps** before Docker job starts
2. **Create docker-context directory** with both binaries
3. **Use Dockerfile.release** instead of Dockerfile
4. **Remove checkout** (not needed, binaries are the context)

### New Dockerfile.release

```dockerfile
# Release Dockerfile - uses pre-built binaries
# For local development, use: make docker (uses Dockerfile)
FROM scratch

ARG TARGETARCH

# Copy pre-built binary for target architecture
COPY mount-monitor-linux-${TARGETARCH} /mount-monitor

# Health check port
EXPOSE 8080

# Non-root user (scratch-compatible numeric UID)
USER 65534

ENTRYPOINT ["/mount-monitor"]
```

### Benefits

1. **Faster builds**: No Go compilation in Docker (~2-3 min saved)
2. **Version consistency**: Binaries have version embedded from build job
3. **Smaller context**: Only binaries, not full source tree
4. **No QEMU overhead**: Pre-built binaries don't need emulation

## Sources

- Docker Multi-Platform GitHub Actions: https://docs.docker.com/build/ci/github-actions/multi-platform/
- Docker Buildx Multi-Platform Guide: https://docs.docker.com/build/building/multi-platform/
- TARGETARCH automatic argument: https://docs.docker.com/reference/dockerfile/#automatic-platform-args-in-the-global-scope
