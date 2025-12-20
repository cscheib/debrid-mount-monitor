# Debrid Mount Monitor

A Kubernetes sidecar container that monitors the health of debrid WebDAV mount points by performing canary file read checks.

## Features

- Canary file health checking with configurable timeout
- Kubernetes-native liveness and readiness probes
- Failure threshold logic to prevent flapping on transient failures
- **Pod restart watchdog** for automatic recovery when mounts become unhealthy
- **Init container mode** for one-shot health gates (block pod startup until mounts are healthy)
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
  ],
  "watchdog": {
    "enabled": true,
    "restartDelay": "0s",
    "maxRetries": 3
  }
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

Most configuration is done via the JSON config file. Only essential runtime flags are provided:

| Flag | Description |
|------|-------------|
| `--config`, `-c` | Path to JSON configuration file |
| `--http-port` | HTTP server port (default: 8080) |
| `--log-level` | Log level: debug, info, warn, error (default: info) |
| `--log-format` | Log format: json, text (default: json) |
| `--init-container-mode` | Run one-shot health check and exit (for Kubernetes init containers) |

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

### Container User ID

The container runs as UID **65534** (`nobody`) by default—a standard non-privileged user suitable for scratch-based images. If your mounted volumes are owned by a different user, the monitor may fail to read the canary file with "permission denied" errors.

**Override at runtime (no rebuild required):**

**Docker:**
```bash
docker run --user 1000:1000 \
  -v /mnt/debrid:/mnt/debrid:ro \
  mount-monitor:latest
```

**Kubernetes:**
```yaml
spec:
  containers:
    - name: mount-monitor
      image: ghcr.io/cscheib/debrid-mount-monitor:latest
      securityContext:
        runAsUser: 1000
        runAsGroup: 1000
```

**Finding the correct UID/GID:**
```bash
# Check ownership of your mounted volume
ls -n /mnt/your-mount
# Output: drwxr-xr-x 2 1000 1000 4096 Dec 20 10:00 .
#                     ^^^^-^^^^-- Use these values
```

See [docs/troubleshooting.md](docs/troubleshooting.md) for diagnosing permission-related health check failures.

## Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /healthz/live` | Liveness probe - returns 200 unless any mount is UNHEALTHY, 503 otherwise |
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

With debug logging:

```bash
docker run -v /mnt/debrid:/mnt/debrid:ro \
  -v $(pwd)/config.json:/app/config.json:ro \
  -p 8080:8080 \
  mount-monitor:latest \
  --log-level=debug
```

### Kubernetes

Deploy with a ConfigMap for configuration. For basic monitoring without watchdog:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mount-monitor-config
data:
  config.json: |
    {
      "mounts": [{"name": "debrid", "path": "/mnt/debrid"}]
    }
---
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: app
    # your main application
  - name: mount-monitor
    image: mount-monitor:latest
    args:
    - --config=/app/config.json
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
    - name: config
      mountPath: /app/config.json
      subPath: config.json
      readOnly: true
  volumes:
  - name: config
    configMap:
      name: mount-monitor-config
```

For watchdog mode (automatic pod restart), see the [Watchdog Mode](#watchdog-mode) section below.

### Init Container Mode

Use `--init-container-mode` to run a one-shot health check that gates pod startup. The monitor checks all mounts once and exits:
- **Exit 0**: All mounts healthy → pod proceeds to main containers
- **Exit 1**: Any mount unhealthy → pod startup blocked

```yaml
apiVersion: v1
kind: Pod
spec:
  initContainers:
    - name: wait-for-mounts
      image: ghcr.io/cscheib/debrid-mount-monitor:latest
      args:
        - --config=/etc/mount-monitor/config.json
        - --init-container-mode
      volumeMounts:
        - name: config
          mountPath: /etc/mount-monitor
        - name: debrid-mount
          mountPath: /mnt/debrid
  containers:
    - name: plex
      image: plexinc/pms-docker:latest
      # ... main container starts only after init container succeeds
```

See [specs/010-init-container-mode/quickstart.md](specs/010-init-container-mode/quickstart.md) for complete documentation including configuration options and troubleshooting.

### Watchdog Mode

Enable watchdog mode for automatic pod restarts when mounts become unhealthy. When a mount fails health checks beyond the failure threshold, the watchdog deletes the pod via the Kubernetes API, triggering a fresh restart with new mount connections.

**When to use watchdog:**
- Your mounts can become stale and require pod restart to recover
- You want automatic recovery without manual intervention
- You're running in Kubernetes and can configure RBAC

**Configuration:**

```json
{
  "mounts": [{"name": "debrid", "path": "/mnt/debrid"}],
  "watchdog": {
    "enabled": true,
    "restartDelay": "0s",
    "maxRetries": 3
  }
}
```

| Option | Description | Default |
|--------|-------------|---------|
| `enabled` | Enable watchdog functionality | `false` |
| `restartDelay` | Delay after mount becomes UNHEALTHY before restart | `0s` |
| `maxRetries` | API retry attempts for pod deletion | `3` |

**Required RBAC resources:**

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: mount-monitor
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: mount-monitor
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "delete"]
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: mount-monitor
subjects:
- kind: ServiceAccount
  name: mount-monitor
roleRef:
  kind: Role
  name: mount-monitor
  apiGroup: rbac.authorization.k8s.io
```

**Pod configuration with watchdog:**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
spec:
  serviceAccountName: mount-monitor
  containers:
  - name: mount-monitor
    image: ghcr.io/cscheib/debrid-mount-monitor:latest
    args:
    - --config=/app/config.json
    env:
    - name: POD_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
    - name: POD_NAMESPACE
      valueFrom:
        fieldRef:
          fieldPath: metadata.namespace
    # ... volume mounts and probes as shown above
```

**State machine:**
1. **HEALTHY** → Mount checks passing
2. **DEGRADED** → Some failures, below threshold
3. **UNHEALTHY** → Failures exceed threshold → watchdog triggers restart

The watchdog gracefully degrades if RBAC permissions are missing or when running outside Kubernetes.

See [docs/troubleshooting.md](docs/troubleshooting.md) for watchdog diagnostics and common issues.

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
