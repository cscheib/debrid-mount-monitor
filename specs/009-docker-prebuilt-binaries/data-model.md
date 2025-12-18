# Data Model: Docker Images with Pre-built Binaries

**Feature**: 009-docker-prebuilt-binaries
**Date**: 2025-12-17

## Overview

This feature is CI/CD infrastructure - no runtime data model. The "data" consists of:
1. Binary artifacts flowing between jobs
2. Docker image layers
3. Multi-arch manifest metadata

## Entities

### Binary Artifact

Represents a compiled executable uploaded by the build job.

| Attribute | Type | Description |
|-----------|------|-------------|
| name | string | Artifact name (e.g., `mount-monitor-linux-amd64`) |
| path | string | File path within artifact (same as name) |
| architecture | enum | `amd64` or `arm64` |
| version | string | Embedded version from build (e.g., `1.2.3`) |
| checksum | string | SHA256 hash (generated in release job) |

**Lifecycle**:
1. Created by build job (matrix strategy)
2. Uploaded to GitHub Actions artifact storage
3. Downloaded by docker job and release job
4. Deleted after workflow retention period

### Docker Build Context

Temporary directory prepared for Docker build.

| Attribute | Type | Description |
|-----------|------|-------------|
| path | string | Working directory path |
| binaries | list | Downloaded binary artifacts |
| dockerfile | string | Path to Dockerfile.release |

**Structure**:
```
docker-context/
├── mount-monitor-linux-amd64   # Downloaded artifact
├── mount-monitor-linux-arm64   # Downloaded artifact
└── Dockerfile.release          # Copied from repo root
```

### Multi-arch Manifest

Docker manifest list created by buildx.

| Attribute | Type | Description |
|-----------|------|-------------|
| tag | string | Image tag (e.g., `v1.2.3`, `latest`) |
| platforms | list | Supported platforms |
| digests | map | Per-platform image digests |

**Example**:
```json
{
  "tag": "v1.2.3",
  "platforms": ["linux/amd64", "linux/arm64"],
  "digests": {
    "linux/amd64": "sha256:abc123...",
    "linux/arm64": "sha256:def456..."
  }
}
```

## Data Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Build Job  │────▶│ Verify Job  │────▶│ Docker Job  │
│  (matrix)   │     │             │     │             │
└─────────────┘     └─────────────┘     └─────────────┘
      │                                        │
      │ upload                                 │ download
      ▼                                        ▼
┌─────────────┐                         ┌─────────────┐
│  Artifacts  │                         │   Context   │
│   Storage   │                         │  Directory  │
└─────────────┘                         └─────────────┘
                                               │
                                               │ buildx
                                               ▼
                                        ┌─────────────┐
                                        │   ghcr.io   │
                                        │  manifest   │
                                        └─────────────┘
```

## State Transitions

### Artifact State

```
┌──────────┐    build     ┌──────────┐   download   ┌──────────┐
│ Building │─────────────▶│ Uploaded │─────────────▶│ In-use   │
└──────────┘              └──────────┘              └──────────┘
                               │                         │
                               │ retention               │ cleanup
                               ▼                         ▼
                          ┌──────────┐              ┌──────────┐
                          │ Expired  │              │ Consumed │
                          └──────────┘              └──────────┘
```

## Validation Rules

1. **Both architectures required**: Docker job must download both amd64 and arm64 binaries
2. **Binary naming convention**: `mount-monitor-linux-{arch}` pattern enforced
3. **Version embedding**: Binaries must have version compiled in (verified in verify job)
4. **Manifest completeness**: Multi-arch manifest must include both platforms
