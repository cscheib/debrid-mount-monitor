# Debrid Mount Monitor

A Kubernetes sidecar container that monitors the health of debrid WebDAV mount points by performing canary file read checks.

## Features

- Canary file health checking with configurable timeout
- Kubernetes-native liveness and readiness probes
- Debounce logic to prevent flapping on transient failures
- Structured JSON logging
- Multi-architecture support (AMD64/ARM64)
- Minimal container image (<20MB)

## Configuration

Configuration can be done via JSON file, environment variables, or CLI flags.

**Precedence** (later overrides earlier): Defaults → Config File → Environment Variables → CLI Flags

### JSON Configuration File

Create a `config.json` file in your working directory or specify a path with `--config`:

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

#### Per-Mount Configuration

Each mount can override global settings:
- `name`: Human-readable identifier (shown in logs and status)
- `path`: Filesystem path to mount point (required)
- `canaryFile`: Override global canary file for this mount
- `failureThreshold`: Override global debounce threshold for this mount

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MOUNT_PATHS` | Comma-separated list of mount paths to monitor | (required) |
| `CANARY_FILE` | Name of the canary file in each mount | `.health-check` |
| `CHECK_INTERVAL` | Interval between health checks | `30s` |
| `READ_TIMEOUT` | Timeout for canary file reads | `5s` |
| `DEBOUNCE_THRESHOLD` | Consecutive failures before unhealthy | `3` |
| `SHUTDOWN_TIMEOUT` | Maximum time for graceful shutdown | `30s` |
| `HTTP_PORT` | Port for health probe endpoints | `8080` |
| `LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |
| `LOG_FORMAT` | Log format (json, text) | `json` |

### CLI Flags

| Flag | Description |
|------|-------------|
| `--config`, `-c` | Path to JSON configuration file |
| `--mount-paths` | Comma-separated list of mount paths |
| `--canary-file` | Canary file name |
| `--check-interval` | Health check interval |
| `--read-timeout` | Canary file read timeout |
| `--debounce-threshold` | Consecutive failures threshold |
| `--shutdown-timeout` | Graceful shutdown timeout |
| `--http-port` | HTTP server port |
| `--log-level` | Log level |
| `--log-format` | Log format |

## Security

### Config File Size Limit

Config files are limited to **1MB maximum** to prevent denial-of-service attacks via excessively large files. This limit is generous—typical configurations are under 10KB. If your config file approaches this limit, consider whether all mounts need to be in a single file.

### File Permissions

For production deployments, secure your config file permissions:

```bash
# Recommended: Owner read/write only
chmod 600 config.json

# Alternative: Owner read/write, group read
chmod 640 config.json
```

The monitor will log a **warning** if the config file is world-writable (`chmod 666` or similar), as this could allow unauthorized modification. This check only runs on Unix systems (Linux, macOS).

## Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /healthz/live` | Liveness probe - always returns 200 if service is running |
| `GET /healthz/ready` | Readiness probe - returns 200 if all mounts healthy, 503 otherwise |
| `GET /healthz/status` | Detailed status of all monitored mounts |

## Usage

### Docker

```bash
docker run -v /mnt/debrid:/mnt/debrid:ro \
  -e MOUNT_PATHS=/mnt/debrid \
  -p 8080:8080 \
  mount-monitor:latest
```

### Kubernetes

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: app
    # your main application
  - name: mount-monitor
    image: mount-monitor:latest
    env:
    - name: MOUNT_PATHS
      value: "/mnt/debrid"
    ports:
    - containerPort: 8080
    livenessProbe:
      httpGet:
        path: /healthz/live
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /healthz/ready
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 10
    volumeMounts:
    - name: debrid-mount
      mountPath: /mnt/debrid
      readOnly: true
```

## Development

```bash
# Run tests
make test

# Build binary
make build

# Build Docker image
make docker

# Run locally
make run
```

## License

MIT
