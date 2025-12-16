# Breaking Changes - Tech Debt Cleanup (Feature 007)

## Summary

This release includes breaking changes to the configuration system. Please review these changes before upgrading.

## Breaking Changes

### 1. Configuration Key Renamed

**Change**: The JSON config key `debounceThreshold` has been renamed to `failureThreshold`.

**Reason**: The term "failure threshold" more accurately describes the consecutive failure count before marking a mount as unhealthy. This aligns with the per-mount `failureThreshold` field.

**Migration**:
```json
// Before
{
  "debounceThreshold": 3
}

// After
{
  "failureThreshold": 3
}
```

### 2. CLI Flag Renamed

**Change**: The CLI flag `--debounce-threshold` has been renamed to `--failure-threshold`.

**Migration**:
```bash
# Before
mount-monitor --debounce-threshold=3

# After
mount-monitor --failure-threshold=3
```

### 3. Most CLI Flags Removed

**Change**: Configuration CLI flags have been removed. Only essential runtime flags remain:
- `--config`, `-c` (kept)
- `--http-port` (kept)
- `--log-level` (kept)
- `--log-format` (kept)

**Removed CLI Flags**:
- `--mount-paths`
- `--canary-file`
- `--check-interval`
- `--read-timeout`
- `--shutdown-timeout`
- `--failure-threshold`

**Reason**: The JSON config file is the primary means of configuration. CLI flags are now reserved for runtime essentials (port, logging) that may need to differ between environments without changing the config file.

**Migration**:
```bash
# Before
mount-monitor --mount-paths=/mnt/debrid --failure-threshold=3

# After - create a config.json
{
  "failureThreshold": 3,
  "mounts": [{"name": "debrid", "path": "/mnt/debrid"}]
}
mount-monitor --config=config.json
```

### 4. Environment Variable Configuration Removed

**Change**: All configuration environment variables have been removed. Configuration must now be done via JSON config file or CLI flags.

**Removed Environment Variables**:
- `MOUNT_PATHS`
- `CANARY_FILE`
- `CHECK_INTERVAL`
- `READ_TIMEOUT`
- `SHUTDOWN_TIMEOUT`
- `DEBOUNCE_THRESHOLD` / `FAILURE_THRESHOLD`
- `HTTP_PORT`
- `LOG_LEVEL`
- `LOG_FORMAT`
- `WATCHDOG_ENABLED`
- `WATCHDOG_RESTART_DELAY`

**NOT Removed** (Kubernetes runtime detection):
- `POD_NAME` - Still used for pod identity via Downward API
- `POD_NAMESPACE` - Still used for pod identity via Downward API
- `KUBERNETES_SERVICE_HOST` - Still used for in-cluster K8s API detection
- `KUBERNETES_SERVICE_PORT` - Still used for in-cluster K8s API detection

**Migration**:

For simple setups, use CLI flags:
```bash
mount-monitor --mount-paths=/mnt/debrid --failure-threshold=3
```

For complex setups, create a `config.json` file:
```json
{
  "failureThreshold": 3,
  "mounts": [
    {"name": "debrid", "path": "/mnt/debrid"}
  ]
}
```

Then mount it in your container:
```yaml
# Kubernetes ConfigMap approach
volumeMounts:
- name: config
  mountPath: /app/config.json
  subPath: config.json
  readOnly: true
volumes:
- name: config
  configMap:
    name: mount-monitor-config
```

### 5. Go Module Namespace Changed

**Change**: The Go module path changed from `github.com/chris/debrid-mount-monitor` to `github.com/cscheib/debrid-mount-monitor`.

**Impact**: This only affects developers importing this module as a library. End users running the container are not affected.

**Migration** (for library users):
```go
// Before
import "github.com/chris/debrid-mount-monitor/internal/health"

// After
import "github.com/cscheib/debrid-mount-monitor/internal/health"
```

## New Features

### Path Configuration Flexibility

Mount paths now explicitly support both absolute and relative paths:
- Absolute: `/mnt/movies`, `/data/media/tv`
- Relative: `./mounts/movies`, `../shared/media`

Relative paths are resolved from the working directory where the monitor runs.

## Configuration Precedence

The new configuration precedence is:

**Defaults → Config File → CLI Flags**

(Environment variables have been removed from the precedence chain)
