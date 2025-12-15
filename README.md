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

Configuration is done via environment variables:

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
