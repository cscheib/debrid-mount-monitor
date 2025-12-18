# Quickstart: Docker Images with Pre-built Binaries

**Feature**: 009-docker-prebuilt-binaries
**Time to implement**: ~30 minutes

## What This Feature Does

Modifies the release workflow so Docker images use pre-compiled binaries from the build job instead of recompiling Go source code. This:
- Eliminates redundant compilation
- Ensures Docker images have correct version embedded
- Reduces CI time

## Prerequisites

- Existing release workflow (`.github/workflows/release.yml`)
- Build job that produces `mount-monitor-linux-amd64` and `mount-monitor-linux-arm64` artifacts

## Implementation Steps

### Step 1: Create Dockerfile.release (2 min)

Copy from `specs/009-docker-prebuilt-binaries/contracts/Dockerfile.release` to repo root:

```dockerfile
FROM scratch
ARG TARGETARCH
COPY mount-monitor-linux-${TARGETARCH} /mount-monitor
EXPOSE 8080
USER 65534
ENTRYPOINT ["/mount-monitor"]
```

### Step 2: Update release.yml Docker Job (5 min)

Replace Job 4 (Docker Build & Push) with the version from `contracts/release-workflow-docker-job.yaml`.

Key changes:
1. Add artifact download steps before Docker build
2. Use sparse checkout for Dockerfile.release only
3. Build from `./docker-context/` directory
4. Use `--file ./docker-context/Dockerfile.release`

### Step 3: Verify Local Development Still Works (2 min)

```bash
# Existing workflow should still work
make docker
docker run mount-monitor:dev --help
```

## Verification

### Manual Test

1. Push a test tag: `git tag v0.0.1-test && git push origin v0.0.1-test`
2. Check GitHub Actions workflow logs:
   - Docker job should NOT show `go build` commands
   - Build time should be faster than before
3. Pull and verify image:
   ```bash
   docker pull ghcr.io/cscheib/debrid-mount-monitor:v0.0.1-test
   docker run ghcr.io/cscheib/debrid-mount-monitor:v0.0.1-test --help
   # Should show version: 0.0.1-test
   ```
4. Delete test tag: `git push origin :refs/tags/v0.0.1-test`

### Success Criteria Checklist

- [ ] Docker build completes without Go compilation
- [ ] `--help` shows correct version in Docker image
- [ ] Both linux/amd64 and linux/arm64 images published
- [ ] `make docker` still works for local development

## Troubleshooting

### Binary not found during Docker build

**Symptom**: `COPY mount-monitor-linux-amd64: file not found`

**Cause**: Artifact download path mismatch

**Fix**: Ensure download path matches build context:
```yaml
- uses: actions/download-artifact@v6
  with:
    name: mount-monitor-linux-amd64
    path: ./docker-context/  # Must match buildx context
```

### Wrong architecture binary copied

**Symptom**: ARM64 image runs amd64 binary (slow, may crash)

**Cause**: TARGETARCH not being used correctly

**Fix**: Verify Dockerfile uses `${TARGETARCH}` in COPY instruction

### Version shows "dev" instead of tag

**Symptom**: `--help` shows `dev` or empty version

**Cause**: Using wrong binaries (not from build job)

**Fix**: Ensure artifact names match exactly:
- Build job uploads: `mount-monitor-linux-amd64`
- Docker job downloads: `mount-monitor-linux-amd64`
