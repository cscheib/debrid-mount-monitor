# Quickstart: JSON Configuration File

**Feature**: 002-json-config
**Date**: 2025-12-14

## Overview

This guide shows how to configure the mount health monitor using a JSON configuration file.

---

## Basic Usage

### 1. Create a Configuration File

Create `config.json` in your working directory:

```json
{
  "mounts": [
    {
      "name": "movies",
      "path": "/mnt/movies"
    },
    {
      "name": "tv",
      "path": "/mnt/tv"
    }
  ]
}
```

### 2. Run the Monitor

```bash
# Uses ./config.json automatically if present
./mount-monitor

# Or specify a custom path
./mount-monitor --config /etc/mount-monitor/config.json
./mount-monitor -c /path/to/config.json
```

---

## Configuration Examples

### Minimal Configuration

```json
{
  "mounts": [
    { "path": "/mnt/media" }
  ]
}
```

### Per-Mount Customization

```json
{
  "checkInterval": "30s",
  "debounceThreshold": 3,
  "mounts": [
    {
      "name": "fast-storage",
      "path": "/mnt/ssd",
      "canaryFile": ".ready",
      "failureThreshold": 2
    },
    {
      "name": "slow-storage",
      "path": "/mnt/hdd",
      "canaryFile": ".health-check",
      "failureThreshold": 5
    }
  ]
}
```

### Full Configuration

```json
{
  "checkInterval": "30s",
  "readTimeout": "5s",
  "shutdownTimeout": "30s",
  "debounceThreshold": 3,
  "httpPort": 8080,
  "logLevel": "info",
  "logFormat": "json",
  "canaryFile": ".health-check",
  "mounts": [
    {
      "name": "movies",
      "path": "/mnt/movies",
      "canaryFile": ".health-check",
      "failureThreshold": 3
    },
    {
      "name": "tv",
      "path": "/mnt/tv",
      "failureThreshold": 5
    }
  ]
}
```

---

## Configuration Precedence

Configuration values are applied in this order (later overrides earlier):

1. **Defaults** (hardcoded in application)
2. **Config File** (`./config.json` or `--config` path)
3. **Environment Variables** (e.g., `CHECK_INTERVAL=60s`)
4. **CLI Flags** (e.g., `--check-interval 60s`)

**Example**: If config.json sets `checkInterval: "30s"` but you run with `CHECK_INTERVAL=60s`, the check interval will be 60 seconds.

---

## Docker / Kubernetes Usage

### Docker Compose

```yaml
services:
  mount-monitor:
    image: mount-monitor:latest
    volumes:
      - ./config.json:/app/config.json:ro
      - /mnt/movies:/mnt/movies:ro
      - /mnt/tv:/mnt/tv:ro
    working_dir: /app
```

### Kubernetes ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mount-monitor-config
data:
  config.json: |
    {
      "checkInterval": "30s",
      "mounts": [
        { "name": "movies", "path": "/mnt/movies" },
        { "name": "tv", "path": "/mnt/tv" }
      ]
    }
---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
        - name: mount-monitor
          volumeMounts:
            - name: config
              mountPath: /app/config.json
              subPath: config.json
      volumes:
        - name: config
          configMap:
            name: mount-monitor-config
```

---

## Validation Errors

The application validates configuration at startup. Common errors:

| Error | Cause | Fix |
|-------|-------|-----|
| `file not found: /path/to/config.json` | Specified file doesn't exist | Check path or remove `--config` flag |
| `parse error at position N` | Invalid JSON syntax | Validate JSON with a linter |
| `mount[0]: missing required field "path"` | Mount without path | Add `"path": "/mnt/..."` |
| `mount[2]: failureThreshold must be >= 1` | Invalid threshold | Set threshold to 1 or higher |
| `empty mounts array` | Config file has no mounts | Add at least one mount or use env vars |

---

## Backwards Compatibility

Existing deployments using environment variables or CLI flags continue to work unchanged:

```bash
# Still works exactly as before
MOUNT_PATHS=/mnt/movies,/mnt/tv ./mount-monitor

# CLI flags still work
./mount-monitor --mount-paths /mnt/movies,/mnt/tv
```

The config file is only used when:
1. `--config` / `-c` flag is provided, OR
2. `./config.json` exists in the working directory

---

## Troubleshooting

### Config File Not Loading

Check startup logs for config source:

```
INFO configuration loaded source=./config.json mounts=2
```

Or if no file was found:

```
INFO configuration loaded source=environment mounts=2
```

### Verifying Per-Mount Settings

Startup logs show each mount's configuration:

```
INFO mount registered name=movies path=/mnt/movies canaryFile=.health-check failureThreshold=3
INFO mount registered name=tv path=/mnt/tv canaryFile=.ready failureThreshold=5
```
