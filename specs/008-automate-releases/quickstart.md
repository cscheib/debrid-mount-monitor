# Quickstart: Creating Releases

**Feature**: 008-automate-releases

This guide explains how to create releases for debrid-mount-monitor once the release workflow is implemented.

---

## Creating a Stable Release

### Step 1: Ensure main branch is ready
```bash
git checkout main
git pull origin main

# Verify CI passes
gh run list --limit 5
```

### Step 2: Create and push a version tag
```bash
# Create annotated tag
git tag -a v1.0.0 -m "Release v1.0.0"

# Push tag to trigger release workflow
git push origin v1.0.0
```

### Step 3: Monitor the release
```bash
# Watch workflow progress
gh run watch

# Or view in browser
gh run view --web
```

### Step 4: Verify release artifacts
After workflow completes (~10-15 minutes):

1. **GitHub Release**: `https://github.com/cscheib/debrid-mount-monitor/releases/tag/v1.0.0`
   - `mount-monitor-linux-amd64`
   - `mount-monitor-linux-arm64`
   - `checksums.txt`
   - Auto-generated release notes

2. **Docker Image**:
   ```bash
   docker pull ghcr.io/cscheib/debrid-mount-monitor:v1.0.0
   docker pull ghcr.io/cscheib/debrid-mount-monitor:latest
   ```

---

## Creating a Pre-Release

Pre-releases use the same process but with a pre-release suffix:

```bash
# Beta release
git tag -a v1.1.0-beta.1 -m "Beta release v1.1.0-beta.1"
git push origin v1.1.0-beta.1

# Release candidate
git tag -a v1.1.0-rc.1 -m "Release candidate v1.1.0-rc.1"
git push origin v1.1.0-rc.1
```

**Pre-release behavior**:
- GitHub Release is marked as "Pre-release"
- Docker `latest` tag is **NOT** updated
- Docker image tagged with version only (e.g., `v1.1.0-beta.1`)

---

## Emergency Manual Release

If you need to create a release without pushing a tag (e.g., re-releasing with fixes):

### Via GitHub UI
1. Go to Actions → Release workflow
2. Click "Run workflow"
3. Enter version (e.g., `v1.0.1`)
4. Click "Run workflow"

### Via CLI
```bash
gh workflow run release.yml -f version=v1.0.1
```

---

## Verifying Downloaded Binaries

Users can verify binary integrity using the checksums file:

```bash
# Download binary and checksums
curl -LO https://github.com/cscheib/debrid-mount-monitor/releases/download/v1.0.0/mount-monitor-linux-amd64
curl -LO https://github.com/cscheib/debrid-mount-monitor/releases/download/v1.0.0/checksums.txt

# Verify checksum
sha256sum -c checksums.txt --ignore-missing
# Expected output: mount-monitor-linux-amd64: OK
```

---

## Version Numbering Convention

Follow [Semantic Versioning](https://semver.org/):

| Version | When to Use |
|---------|-------------|
| `v1.0.0` | First stable release |
| `v1.0.1` | Bug fixes only |
| `v1.1.0` | New features (backward compatible) |
| `v2.0.0` | Breaking changes |
| `v1.1.0-beta.1` | Beta testing |
| `v1.1.0-rc.1` | Release candidate |

---

## Troubleshooting

### Release workflow failed
```bash
# View failed run
gh run view --failed

# Re-run failed jobs only
gh run rerun <run-id> --failed
```

### Tag already exists
```bash
# Delete local tag
git tag -d v1.0.0

# Delete remote tag (caution: cannot undo if release exists)
git push origin :refs/tags/v1.0.0
```

### Docker push failed
The workflow automatically retries Docker push up to 3 times. If still failing:
1. Check ghcr.io status
2. Re-run the workflow manually
3. Verify repository permissions allow package writes
