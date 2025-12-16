# Debrid Mount Monitor

A Kubernetes sidecar container that monitors the health of debrid WebDAV mount points by performing canary file read checks.

## Features

- Canary file health checking with configurable timeout
- Kubernetes-native liveness and readiness probes
- Failure threshold logic to prevent flapping on transient failures
- Structured JSON logging
- Multi-architecture support (AMD64/ARM64)
- Minimal container image (<20MB)

## Configuration

Configuration can be done via JSON file or CLI flags.

**Precedence** (later overrides earlier): Defaults → Config File → CLI Flags

### JSON Configuration File

Create a `config.json` file in your working directory or specify a path with `--config`:

```json
{
  "checkInterval": "30s",
  "readTimeout": "5s",
  "shutdownTimeout": "30s",
  "failureThreshold": 3,
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
    },
    {
      "name": "local-data",
      "path": "./data/local"
    }
  ]
}
```

#### Per-Mount Configuration

Each mount can override global settings:
- `name`: Human-readable identifier (shown in logs and status)
- `path`: Filesystem path to mount point (required) - can be absolute or relative
- `canaryFile`: Override global canary file for this mount (always relative to mount path)
- `failureThreshold`: Override global failure threshold for this mount

#### Path Configuration

**Mount Paths:** Mount paths can be specified as either absolute or relative paths:
- Absolute: `/mnt/movies`, `/data/media/tv`
- Relative: `./mounts/movies`, `../shared/media`

Relative paths are resolved from the working directory where the monitor runs.

**Canary Files:** Canary file paths are always relative to their respective mount path. For example, with a mount at `/mnt/movies` and canary file `.health-check`, the full canary path is `/mnt/movies/.health-check`.

### CLI Flags

| Flag | Description |
|------|-------------|
| `--config`, `-c` | Path to JSON configuration file |
| `--mount-paths` | Comma-separated list of mount paths |
| `--canary-file` | Canary file name |
| `--check-interval` | Health check interval |
| `--read-timeout` | Canary file read timeout |
| `--failure-threshold` | Consecutive failures threshold |
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
  -v $(pwd)/config.json:/app/config.json:ro \
  -p 8080:8080 \
  mount-monitor:latest
```

Or using CLI flags:

```bash
docker run -v /mnt/debrid:/mnt/debrid:ro \
  -p 8080:8080 \
  mount-monitor:latest \
  --mount-paths=/mnt/debrid
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
    args:
    - --mount-paths=/mnt/debrid
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

For more complex configurations, mount a ConfigMap containing `config.json`:

```yaml
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

### Local Kubernetes Development

For testing in a real Kubernetes environment locally, we provide KIND (Kubernetes IN Docker) support:

```bash
# Create a local KIND cluster
make kind-create

# Build and load the image into KIND
make kind-load

# Deploy the monitor as a sidecar
make kind-deploy

# View logs
make kind-logs

# Rebuild and redeploy after code changes
make kind-redeploy

# Run automated e2e watchdog test
make kind-test

# Clean up
make kind-delete
```

**Custom Namespace:** Deploy to a custom namespace:
```bash
KIND_NAMESPACE=my-namespace make kind-deploy
```

See [deploy/kind/README.md](deploy/kind/README.md) for detailed documentation on:
- Simulating mount failures
- Verifying probe behavior
- Quick iteration workflow
- Environment variables (`KIND_CLUSTER_NAME`, `KIND_NAMESPACE`, `KEEP_CLUSTER`)

See [docs/troubleshooting.md](docs/troubleshooting.md) for:
- Common issues and resolutions
- RBAC troubleshooting
- Debug logging
- Diagnostic commands

## License

MIT
