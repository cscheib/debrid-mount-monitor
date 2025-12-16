# Use Cases

This document describes the primary use cases for debrid-mount-monitor, organized by user persona and feature.

## Product Overview

**Debrid Mount Monitor** is a Kubernetes-native sidecar container that ensures WebDAV mount points (particularly debrid storage services like Real-Debrid, AllDebrid, Premiumize) remain healthy and accessible. It provides automatic detection and recovery from mount failures.

## Target User Personas

### Home Lab Enthusiast
- Runs Plex/Jellyfin with debrid-backed storage
- Uses Kubernetes (k3s, microk8s) for home media automation
- Wants "set and forget" reliability for streaming

### Media Server Administrator
- Manages *arr stack (Radarr, Sonarr, Lidarr, etc.)
- Relies on rclone/WebDAV mounts for debrid integration
- Needs visibility into mount health across multiple services

### DevOps Engineer
- Operates Kubernetes clusters with NFS/WebDAV mounts
- Needs automated recovery without manual intervention
- Requires audit trails and observability

---

## Core Use Cases

### UC-1: Automatic Pod Recovery from Stale Mounts

**Personas:** Home Lab Enthusiast, Media Server Admin
**Feature:** Pod Restart Watchdog

**Scenario:**
A Plex server pod mounts debrid storage via rclone WebDAV. The debrid service experiences a brief outage, causing the mount to become stale. Without intervention, Plex shows empty libraries or hangs on playback.

**How debrid-mount-monitor helps:**
1. Sidecar continuously checks canary file on the mount (every 30s by default)
2. After 3 consecutive failures (failure threshold), mount is marked UNHEALTHY
3. Watchdog automatically deletes the pod via Kubernetes API
4. ReplicaSet creates fresh pod with fresh mount connection
5. Plex comes back online with working storage

**Value:** Zero-touch recovery from mount failures. No more waking up to broken Plex.

---

### UC-2: Preventing Traffic to Pods with Unhealthy Mounts

**Personas:** DevOps Engineer, Media Server Admin
**Feature:** Kubernetes Readiness Probes (`/healthz/ready`)

**Scenario:**
A Radarr pod is part of a Deployment. The mount becomes degraded but not fully failed. Users accessing Radarr see errors or missing content.

**How debrid-mount-monitor helps:**
1. Readiness probe returns 503 when any mount is not HEALTHY
2. Kubernetes removes pod from Service endpoints
3. Traffic routes only to healthy replicas
4. Degraded pod has time to recover or gets restarted

**Value:** Users never see broken UI from mount issues.

---

### UC-3: Graceful Handling of Transient Network Issues

**Personas:** All
**Feature:** Failure Threshold Logic with Configurable Thresholds

**Scenario:**
Network has occasional hiccups. A single failed health check shouldn't trigger a pod restart, as this would cause unnecessary downtime.

**How debrid-mount-monitor helps:**
1. Single failure transitions mount to DEGRADED (not UNHEALTHY)
2. Failure counter increments on each consecutive failure
3. Only after threshold (default: 3) does mount become UNHEALTHY
4. If mount recovers before threshold, counter resets
5. No unnecessary restarts from transient issues

**Configuration example:**
```json
{
  "mounts": [
    {
      "path": "/mnt/debrid",
      "failureThreshold": 5
    }
  ]
}
```

**Value:** Stability without sacrificing responsiveness.

---

### UC-4: Multi-Mount Monitoring for Complex Media Stacks

**Personas:** Media Server Admin
**Feature:** Per-Mount Configuration

**Scenario:**
A pod runs Sonarr with multiple mounts:
- `/mnt/movies` - Real-Debrid (stable)
- `/mnt/tv` - AllDebrid (less stable)
- `/mnt/downloads` - Local NFS (very stable)

Each mount has different reliability characteristics.

**Configuration example:**
```json
{
  "mounts": [
    { "path": "/mnt/movies", "failureThreshold": 3, "canaryFile": ".rd-health" },
    { "path": "/mnt/tv", "failureThreshold": 5, "canaryFile": ".ad-health" },
    { "path": "/mnt/downloads", "failureThreshold": 2, "canaryFile": ".nfs-health" }
  ]
}
```

**Value:** Fine-tuned monitoring per mount's reliability profile.

---

### UC-5: Visibility into Mount Health for Troubleshooting

**Personas:** DevOps Engineer, Media Server Admin
**Feature:** Status Endpoint (`/healthz/status`)

**Scenario:**
Users report intermittent issues. Admin needs to understand mount health status.

**Usage:**
```bash
curl http://localhost:8080/healthz/status
```

**Response:**
```json
{
  "timestamp": "2025-12-16T10:30:00Z",
  "mounts": [
    {
      "path": "/mnt/debrid",
      "name": "debrid-storage",
      "status": "DEGRADED",
      "lastCheck": "2025-12-16T10:29:55Z",
      "failureCount": 2,
      "lastError": "read /mnt/debrid/.health-check: i/o timeout"
    }
  ]
}
```

**Value:** Quick diagnostics without log diving.

---

### UC-6: Audit Trail for Compliance and Debugging

**Personas:** DevOps Engineer
**Feature:** Kubernetes Events + Structured Logging

**Scenario:**
Post-incident review needs to understand why pods restarted overnight.

**How debrid-mount-monitor helps:**

1. WatchdogRestart events created in Kubernetes:
   ```bash
   kubectl get events --field-selector reason=WatchdogRestart
   ```

2. Structured JSON logs with full context:
   ```json
   {
     "level": "WARN",
     "msg": "mount unhealthy, triggering pod restart",
     "mount": "/mnt/debrid",
     "failureCount": 3,
     "unhealthyDuration": "1m30s"
   }
   ```

**Value:** Full audit trail for incident response.

---

### UC-7: Integration with Existing Monitoring Stack

**Personas:** DevOps Engineer
**Feature:** HTTP Health Endpoints + JSON Logging

**Scenario:**
Organization uses Prometheus + Grafana for monitoring. Need to integrate mount health metrics.

**Integration points:**
- Prometheus blackbox exporter for endpoint monitoring
- Loki/Elasticsearch for log aggregation
- Kubernetes events for alertmanager

**How to integrate:**
1. Scrape `/healthz/status` endpoint for mount status
2. Parse JSON logs for structured metrics
3. Alert on DEGRADED/UNHEALTHY state transitions
4. Visualize mount health trends over time

**Value:** Fits into existing observability infrastructure.

---

### UC-8: Local Development and Testing

**Personas:** DevOps Engineer, Contributors
**Feature:** KIND-based Local Dev Environment

**Scenario:**
Engineer wants to test mount failure behavior before deploying to production.

**Usage:**
```bash
make kind-create      # Create local K8s cluster
make kind-deploy      # Deploy with test configuration
make kind-test        # Run automated e2e test

# Simulate failure manually
kubectl exec $POD -c main-app -- rm /mnt/test/.health-check

# Watch recovery
make kind-logs
```

**Value:** Safe testing environment that mirrors production behavior.

---

### UC-9: Minimal Resource Footprint

**Personas:** Home Lab Enthusiast
**Feature:** Scratch-based Container (<20MB)

**Scenario:**
Running on a Raspberry Pi or low-spec NAS with limited resources.

**Specifications:**
- Container image under 20MB
- Single Go binary with no external dependencies
- Minimal memory footprint (health checks are lightweight)
- Multi-architecture support (AMD64 + ARM64)

**Value:** Runs anywhere without resource concerns.

---

### UC-10: Flexible Deployment Options

**Personas:** All
**Feature:** JSON Config File with Runtime Overrides

**Scenario:**
Configuration via JSON file with essential runtime flags:
- All mount and timing configuration: JSON config file
- Runtime overrides: `--http-port`, `--log-level`, `--log-format`

**Examples:**

```json
// config.json
{
  "checkInterval": "5s",
  "mounts": [{"name": "test", "path": "/mnt/test"}]
}
```

```bash
# Development with debug logging
./mount-monitor --config=config.json --log-level=debug --log-format=text
```

```yaml
# Kubernetes Production - ConfigMap with JSON config
volumeMounts:
  - name: config
    mountPath: /app/config.json
    subPath: config.json
```

**Value:** Clean separation between configuration (JSON) and runtime settings (flags).

---

## Use Case Status

| Use Case | Business Value | Status |
|----------|---------------|--------|
| UC-1: Auto Recovery | High | Complete |
| UC-2: Readiness Probes | High | Complete |
| UC-3: Failure Threshold | High | Complete |
| UC-4: Multi-Mount | Medium | Complete |
| UC-5: Status Endpoint | Medium | Complete |
| UC-6: Audit Trail | Medium | Complete |
| UC-7: Monitoring Integration | Medium | Complete |
| UC-8: Local Dev | Medium | Complete |
| UC-9: Minimal Footprint | Low | Complete |
| UC-10: Flexible Config | Low | Complete |

---

## Future Considerations

The following use cases have been identified as potential future enhancements:

| Use Case | Description |
|----------|-------------|
| Prometheus Metrics | Dedicated `/metrics` endpoint with Prometheus format |
| Notifications | Slack/Discord webhook integration for mount failures |
| Multi-Pod Coordination | Coordinate restarts across replicas to prevent simultaneous downtime |
| Mount Repair | Attempt remounting before triggering pod restart |
| Web Dashboard | Simple UI for viewing mount health across cluster |

---

## See Also

- [README.md](../README.md) - Getting started and quick reference
- [Troubleshooting](troubleshooting.md) - Common issues and solutions
- [Configuration Examples](../bin/) - Sample configuration files
