# Quickstart: Mount Health Monitor

**Date**: 2025-12-14
**Feature**: 001-mount-health-monitor

## Prerequisites

- Go 1.21 or later
- Docker (for container builds)
- A mount point to monitor (for testing)

## Build

### Local Binary

```bash
# Build for current platform
go build -o mount-monitor ./cmd/mount-monitor

# Build static binary for Linux (container deployment)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mount-monitor ./cmd/mount-monitor
```

### Container Image

```bash
# Build multi-architecture image
docker buildx build --platform linux/amd64,linux/arm64 -t mount-monitor:latest .

# Build for local testing (single arch)
docker build -t mount-monitor:latest .
```

## Configuration

All configuration via environment variables or command-line flags:

| Environment Variable | Flag | Default | Description |
|---------------------|------|---------|-------------|
| `MOUNT_PATHS` | `--mount-paths` | (required) | Comma-separated paths to monitor |
| `CANARY_FILE` | `--canary-file` | `.health-check` | File to read within each mount |
| `CHECK_INTERVAL` | `--check-interval` | `30s` | Time between checks |
| `READ_TIMEOUT` | `--read-timeout` | `5s` | Timeout for file read |
| `DEBOUNCE_THRESHOLD` | `--debounce-threshold` | `3` | Failures before unhealthy |
| `HTTP_PORT` | `--http-port` | `8080` | Port for health endpoints |
| `LOG_LEVEL` | `--log-level` | `info` | debug/info/warn/error |
| `LOG_FORMAT` | `--log-format` | `json` | json/text |

## Run Locally

### 1. Create a test mount with canary file

```bash
mkdir -p /tmp/test-mount
echo "ok" > /tmp/test-mount/.health-check
```

### 2. Start the monitor

```bash
# Using environment variables
export MOUNT_PATHS=/tmp/test-mount
export LOG_FORMAT=text
./mount-monitor

# Or using flags
./mount-monitor --mount-paths=/tmp/test-mount --log-format=text
```

### 3. Test the endpoints

```bash
# Check liveness probe
curl -i http://localhost:8080/healthz/live

# Check readiness probe
curl -i http://localhost:8080/healthz/ready
```

### 4. Simulate failure

```bash
# Remove the canary file
rm /tmp/test-mount/.health-check

# Wait for check interval + debounce (default: 30s * 3 = 90s for liveness)
# Readiness will fail immediately on next check

# Watch the logs and probe responses
curl -i http://localhost:8080/healthz/ready  # Should return 503
```

### 5. Simulate recovery

```bash
# Restore the canary file
echo "ok" > /tmp/test-mount/.health-check

# Wait for next check interval
curl -i http://localhost:8080/healthz/ready  # Should return 200
```

## Run in Kubernetes

### 1. Create ConfigMap (optional, for canary file creation)

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mount-monitor-config
data:
  MOUNT_PATHS: "/mnt/debrid"
  CHECK_INTERVAL: "30s"
  DEBOUNCE_THRESHOLD: "3"
```

### 2. Deploy as sidecar

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: media-server
spec:
  containers:
    # Main application container
    - name: plex
      image: plexinc/pms-docker:latest
      volumeMounts:
        - name: media
          mountPath: /mnt/debrid

    # Health monitor sidecar
    - name: mount-monitor
      image: ghcr.io/your-org/mount-monitor:latest
      env:
        - name: MOUNT_PATHS
          value: "/mnt/debrid"
        - name: CANARY_FILE
          value: ".health-check"
      volumeMounts:
        - name: media
          mountPath: /mnt/debrid
          readOnly: true
      livenessProbe:
        httpGet:
          path: /healthz/live
          port: 8080
        initialDelaySeconds: 10
        periodSeconds: 10
        failureThreshold: 1  # Sidecar handles debounce, K8s can fail fast
      readinessProbe:
        httpGet:
          path: /healthz/ready
          port: 8080
        initialDelaySeconds: 5
        periodSeconds: 5
        failureThreshold: 1

  volumes:
    - name: media
      persistentVolumeClaim:
        claimName: debrid-pvc
```

## Verify Behavior

### Expected Log Output (JSON format)

```json
{"time":"2025-12-14T10:00:00Z","level":"INFO","msg":"starting mount monitor","version":"1.0.0"}
{"time":"2025-12-14T10:00:00Z","level":"INFO","msg":"mount registered","path":"/mnt/debrid","canary":"/mnt/debrid/.health-check"}
{"time":"2025-12-14T10:00:00Z","level":"INFO","msg":"http server started","port":8080}
{"time":"2025-12-14T10:00:05Z","level":"INFO","msg":"health check completed","path":"/mnt/debrid","status":"healthy","duration_ms":2}
{"time":"2025-12-14T10:00:35Z","level":"WARN","msg":"health check failed","path":"/mnt/debrid","error":"read timeout","failure_count":1}
{"time":"2025-12-14T10:01:05Z","level":"WARN","msg":"health check failed","path":"/mnt/debrid","error":"read timeout","failure_count":2}
{"time":"2025-12-14T10:01:35Z","level":"ERROR","msg":"mount unhealthy","path":"/mnt/debrid","failure_count":3,"previous_state":"degraded","new_state":"unhealthy"}
```

### Expected Probe Responses

**Healthy state:**
```bash
$ curl -s http://localhost:8080/healthz/live | jq
{
  "status": "healthy",
  "timestamp": "2025-12-14T10:00:05Z",
  "mounts": [
    {
      "path": "/mnt/debrid",
      "status": "healthy",
      "last_check": "2025-12-14T10:00:05Z",
      "failure_count": 0
    }
  ]
}
```

**Unhealthy state:**
```bash
$ curl -s http://localhost:8080/healthz/live | jq
{
  "status": "unhealthy",
  "timestamp": "2025-12-14T10:01:35Z",
  "mounts": [
    {
      "path": "/mnt/debrid",
      "status": "unhealthy",
      "last_check": "2025-12-14T10:01:35Z",
      "failure_count": 3,
      "error": "read timeout: context deadline exceeded"
    }
  ]
}
```

## Troubleshooting

| Symptom | Cause | Solution |
|---------|-------|----------|
| Probe always returns 503 | Canary file missing | Create `.health-check` file in mount |
| Probe returns 503 immediately | Readiness fails on any failure | This is expected; check mount accessibility |
| High memory usage | Unlikely with this design | Check for goroutine leaks in logs |
| Slow probe response | Mount is stale/hung | Read timeout will trigger after configured duration |
