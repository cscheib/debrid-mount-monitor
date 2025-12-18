# Quickstart: Init Container Mode

## Overview

Init container mode runs the mount monitor as a one-shot health gate. It checks all configured mounts once and exits immediately with:
- **Exit 0**: All mounts healthy (pod can proceed)
- **Exit 1**: One or more mounts unhealthy (pod startup blocked)

## Usage

```bash
# Basic usage with config file
./mount-monitor --config /etc/mount-monitor/config.json --init-container-mode

# With logging options
./mount-monitor -c /etc/mount-monitor/config.json --init-container-mode --log-format text
```

## Kubernetes Deployment

Use as an initContainer to gate pod startup on mount availability:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: media-server
spec:
  initContainers:
    - name: wait-for-mounts
      image: ghcr.io/cscheib/debrid-mount-monitor:latest
      args:
        - --config
        - /etc/mount-monitor/config.json
        - --init-container-mode
      volumeMounts:
        - name: config
          mountPath: /etc/mount-monitor
        - name: debrid-movies
          mountPath: /mnt/movies
        - name: debrid-tv
          mountPath: /mnt/tv
  containers:
    - name: plex
      image: plexinc/pms-docker:latest
      volumeMounts:
        - name: debrid-movies
          mountPath: /mnt/movies
        - name: debrid-tv
          mountPath: /mnt/tv
  volumes:
    - name: config
      configMap:
        name: mount-monitor-config
    - name: debrid-movies
      # ... your mount configuration
    - name: debrid-tv
      # ... your mount configuration
```

## Configuration

Uses the same `config.json` format as the standard monitor mode. Example:

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
  ],
  "canaryFile": ".health-check",
  "readTimeout": "5s"
}
```

### Relevant Settings

| Setting | Purpose in Init Mode |
|---------|---------------------|
| `mounts` | List of mounts to check (required) |
| `canaryFile` | File to read for health check (default: `.health-check`) |
| `readTimeout` | Timeout for each mount check (default: 5s) |
| `logLevel` | Log verbosity (default: info) |
| `logFormat` | Log format: json or text (default: json) |

### Ignored Settings

These settings are not used in init mode (validation is skipped):
- `checkInterval` - only one check performed
- `httpPort` - no HTTP server started
- `shutdownTimeout` - immediate exit
- `watchdog.*` - watchdog not started

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All mount checks passed |
| 1 | One or more checks failed, or configuration error |

## Log Output

### Success (JSON format)
```json
{"time":"2025-12-17T10:00:00Z","level":"INFO","msg":"mount check passed","name":"movies","path":"/mnt/movies","duration":"2ms"}
{"time":"2025-12-17T10:00:00Z","level":"INFO","msg":"mount check passed","name":"tv","path":"/mnt/tv","duration":"3ms"}
{"time":"2025-12-17T10:00:00Z","level":"INFO","msg":"all mount checks passed","count":2}
```

### Failure (JSON format)
```json
{"time":"2025-12-17T10:00:00Z","level":"INFO","msg":"mount check passed","name":"movies","path":"/mnt/movies","duration":"2ms"}
{"time":"2025-12-17T10:00:05Z","level":"ERROR","msg":"mount check failed","name":"tv","path":"/mnt/tv","error":"context deadline exceeded","duration":"5s"}
{"time":"2025-12-17T10:00:05Z","level":"ERROR","msg":"one or more mount checks failed","count":2}
```

## Troubleshooting

### Check exits with code 1 but no error in logs

Ensure the canary file exists in each mount:
```bash
# Create canary files
echo "healthy" > /mnt/movies/.health-check
echo "healthy" > /mnt/tv/.health-check
```

### Timeout errors

Increase `readTimeout` in config if mounts are slow (e.g., remote NFS):
```json
{
  "readTimeout": "30s"
}
```

### Pod stuck in Init state

Check init container logs:
```bash
kubectl logs <pod-name> -c wait-for-mounts
```
