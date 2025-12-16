# Quickstart: Tech Debt Cleanup Implementation

**Feature**: 007-tech-debt-cleanup
**Date**: 2025-12-16

## Prerequisites

- Go 1.21+ installed
- Repository cloned and on branch `007-tech-debt-cleanup`

## Implementation Order

Execute these changes in order to minimize broken states:

### Step 1: Namespace Rename (FR-001, FR-002, FR-003)

```bash
# Update go.mod
go mod edit -module github.com/cscheib/debrid-mount-monitor

# Update all imports in .go files
find . -name "*.go" -exec sed -i '' 's|github.com/chris/debrid-mount-monitor|github.com/cscheib/debrid-mount-monitor|g' {} \;

# Verify build succeeds
go build ./...
go test ./...
```

### Step 2: Terminology Alignment (FR-004 through FR-008)

Files to modify:
1. `internal/config/config.go` - struct field, flag definition, default, validation
2. `internal/config/file.go` - JSON struct field
3. `internal/health/state.go` - comments and function parameter names
4. `internal/monitor/monitor.go` - variable names
5. `internal/server/server.go` - comments
6. `cmd/mount-monitor/main.go` - log field name
7. `README.md` - all documentation references
8. All test files - update variable names and assertions

Key replacements:
- `DebounceThreshold` → `FailureThreshold` (struct fields)
- `debounceThreshold` → `failureThreshold` (JSON, variables, log keys)
- `--debounce-threshold` → `--failure-threshold` (CLI flag)
- "debounce threshold" → "failure threshold" (comments, docs)
- "debounce" → "failure" in context of threshold descriptions

### Step 3: Remove Environment Variables (FR-011 through FR-022)

In `internal/config/config.go`, remove the entire block of `os.Getenv` calls (~lines 112-154):
- `MOUNT_PATHS`
- `CANARY_FILE`
- `CHECK_INTERVAL`
- `READ_TIMEOUT`
- `SHUTDOWN_TIMEOUT`
- `DEBOUNCE_THRESHOLD` (already renamed in Step 2)
- `HTTP_PORT`
- `LOG_LEVEL`
- `LOG_FORMAT`
- `WATCHDOG_ENABLED`
- `WATCHDOG_RESTART_DELAY`

**Do NOT remove** (these are K8s runtime detection, FR-023):
- `KUBERNETES_SERVICE_HOST` in `internal/watchdog/k8s_client.go`
- `KUBERNETES_SERVICE_PORT` in `internal/watchdog/k8s_client.go`
- `POD_NAME` in `cmd/mount-monitor/main.go`
- `POD_NAMESPACE` in `cmd/mount-monitor/main.go`

Remove corresponding tests in `tests/unit/config_file_test.go`.

### Step 4: Documentation Updates (FR-009, FR-010, FR-022)

Update `README.md`:
1. Remove "Environment Variables" section entirely
2. Update "Configuration" precedence line (remove env vars from chain)
3. Add path documentation to JSON Configuration section:
   - Mount paths can be relative or absolute
   - Canary files are always relative to mount path
4. Update JSON example to show both path formats
5. Remove env var references from Docker/Kubernetes examples

### Step 5: Verify All Changes

```bash
# Build
go build ./...

# Run all tests
go test ./...

# Verify no debounce references remain (should return empty or only CHANGELOG entries)
grep -ri "debounce" --include="*.go" --include="*.md" .

# Verify no old namespace references
grep -r "github.com/chris/" --include="*.go" .

# Verify env var removal (should only show K8s runtime vars)
grep -r "os.Getenv" --include="*.go" .
```

## Testing Checklist

- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes all tests
- [ ] `go mod tidy` makes no changes
- [ ] No "debounce" in user-facing config/docs/logs
- [ ] No config env var reads (only K8s runtime vars)
- [ ] README shows correct config key names
- [ ] JSON example shows relative and absolute paths

## Breaking Change Documentation

Add to CHANGELOG.md or release notes:

```markdown
## Breaking Changes

### Configuration

- **Renamed**: JSON config key `debounceThreshold` is now `failureThreshold`
- **Removed**: Environment variable configuration support

**Migration**:
1. Update `config.json`: change `debounceThreshold` to `failureThreshold`
2. If using environment variables, migrate to JSON config file or CLI flags
```
