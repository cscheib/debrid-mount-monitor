# Research: Automate Releases

**Feature**: 008-automate-releases
**Date**: 2025-12-17

## Summary

Research findings for implementing a GitHub Actions release workflow for the debrid-mount-monitor project.

---

## 1. GitHub Release Action Selection

### Decision: `softprops/action-gh-release@v2`

### Rationale
- Actively maintained (10k+ GitHub stars)
- Simple API for uploading multiple binary assets
- Built-in `generate_release_notes: true` option
- Supports pre-release flag via `prerelease:` input
- Compatible with `workflow_dispatch` for manual triggers
- Already used widely in Go ecosystem

### Alternatives Considered

| Action | Status | Why Not Chosen |
|--------|--------|----------------|
| `actions/create-release` | DEPRECATED | No longer maintained, uses vulnerable Node.js version |
| `ncipollo/release-action` | Active | More complex API; overkill for our simple use case |
| GoReleaser | Active | Full release manager; adds unnecessary complexity for single binary |

### Usage Pattern
```yaml
- uses: softprops/action-gh-release@v2
  with:
    files: |
      bin/mount-monitor-linux-amd64
      bin/mount-monitor-linux-arm64
      checksums.txt
    generate_release_notes: true
    prerelease: ${{ env.IS_PRERELEASE }}
```

---

## 2. Docker Publishing to ghcr.io

### Decision: Use existing `docker/build-push-action@v6` with `docker/login-action@v3`

### Rationale
- Already used in CI workflow (proven pattern)
- Native multi-arch support via `platforms: linux/amd64,linux/arm64`
- GITHUB_TOKEN authentication (no PAT required for same-repo)
- Automatic manifest creation for multi-arch images

### Tagging Strategy
```yaml
tags: |
  ghcr.io/${{ github.repository }}:${{ github.ref_name }}
  ghcr.io/${{ github.repository }}:latest  # Only for stable releases
```

### Authentication
```yaml
- uses: docker/login-action@v3
  with:
    registry: ghcr.io
    username: ${{ github.actor }}
    password: ${{ secrets.GITHUB_TOKEN }}
```

### Permissions Required
```yaml
permissions:
  contents: write   # For creating releases
  packages: write   # For pushing to ghcr.io
```

---

## 3. Retry Mechanism

### Decision: `nick-fields/retry@v3` action

### Rationale
- Purpose-built for GitHub Actions retry scenarios
- Configurable timeout, max attempts, and wait time
- Cleaner than shell loops or `continue-on-error` hacks
- Widely adopted (handles transient registry failures gracefully)

### Usage Pattern (for Docker push)
```yaml
- name: Push Docker image with retry
  uses: nick-fields/retry@v3
  with:
    timeout_minutes: 10
    max_attempts: 3
    retry_wait_seconds: 30
    command: |
      docker buildx build --push ...
```

### Alternative Considered
Manual `continue-on-error` with conditional retries—rejected because it's verbose and harder to maintain.

---

## 4. Pre-Release Detection

### Decision: Shell script regex check

### Rationale
- Simple, no additional action dependency
- SemVer spec: pre-release versions contain `-` after patch number
- Examples: `v1.0.0-beta.1`, `v1.0.0-rc.1`, `v1.0.0-alpha`

### Implementation
```yaml
- name: Detect pre-release
  run: |
    VERSION="${{ github.ref_name }}"
    if [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+-.+$ ]]; then
      echo "IS_PRERELEASE=true" >> $GITHUB_ENV
    else
      echo "IS_PRERELEASE=false" >> $GITHUB_ENV
    fi
```

### Usage in Subsequent Steps
```yaml
prerelease: ${{ env.IS_PRERELEASE == 'true' }}
```

### Latest Tag Logic
Only update `latest` when `IS_PRERELEASE == 'false'`:
```yaml
tags: |
  ghcr.io/${{ github.repository }}:${{ github.ref_name }}
  ${{ env.IS_PRERELEASE == 'false' && format('ghcr.io/{0}:latest', github.repository) || '' }}
```

---

## 5. Checksum Generation

### Decision: Manual `sha256sum` command

### Rationale
- No additional action dependency
- Standard Unix tool available on all GitHub runners
- Full control over output format
- Simple to implement and understand

### Implementation
```yaml
- name: Generate checksums
  run: |
    cd bin
    sha256sum mount-monitor-* > checksums.txt
    cat checksums.txt
```

### Output Format (checksums.txt)
```
a1b2c3d4e5...  mount-monitor-linux-amd64
f6g7h8i9j0...  mount-monitor-linux-arm64
```

### Alternative Considered
`jmgilman/actions-generate-checksum`—rejected because manual approach is simpler and avoids another dependency.

---

## 6. Version Injection

### Decision: Use existing Makefile LDFLAGS pattern

### Rationale
- Already implemented: `LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"`
- Strip `v` prefix from tag for clean version string
- Consistent with existing build process

### Implementation
```yaml
- name: Build binaries
  run: |
    VERSION="${GITHUB_REF_NAME#v}"  # Strip 'v' prefix
    make build-all VERSION=$VERSION
```

---

## 7. Workflow Structure

### Decision: Sequential jobs with dependencies

### Rationale
- Test must pass before any release artifacts are created (FR-012)
- Binary builds can run in parallel via matrix
- Docker build depends on successful binary verification
- Release creation is final step after all artifacts ready

### Job Flow
```
test (gate) → build-binaries (matrix) → verify-binaries → build-docker → release
```

### Manual Dispatch Support
```yaml
on:
  push:
    tags:
      - 'v*'
  workflow_dispatch:
    inputs:
      version:
        description: 'Version to release (e.g., v1.0.0)'
        required: true
```

---

## References

- [softprops/action-gh-release](https://github.com/softprops/action-gh-release)
- [docker/build-push-action](https://github.com/docker/build-push-action)
- [docker/login-action](https://github.com/docker/login-action)
- [nick-fields/retry](https://github.com/nick-fields/retry)
- [GitHub Container Registry Docs](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [SemVer Specification](https://semver.org/)
