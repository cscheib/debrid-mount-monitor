# Feature Specification: Mount Health Monitor

**Feature Branch**: `001-mount-health-monitor`
**Created**: 2025-12-14
**Status**: Draft
**Input**: User description: "A service that monitors debrid WebDAV mount health, restarts dependent services when unhealthy, and gates service startup until mounts are verified healthy"

## Clarifications

### Session 2025-12-14

- Q: How should the monitor trigger restarts of dependent services? → A: Kubernetes manages pod lifecycle. Monitor runs as sidecar and fails its Kubernetes health check (liveness/readiness probe) when mounts are unhealthy, causing Kubernetes to restart the pod.
- Q: How should the monitor detect that a mount is stale or unresponsive? → A: Read canary file - attempt to read a small test file within the mount with timeout. More robust than stat-only checks as it verifies the mount is actually serving data.
- Q: What timeout should be used for the canary file read operation? → A: Configurable with sensible default (e.g., 5 seconds). Operator can tune per environment.
- Q: What CI/CD system should be used for building and testing? → A: GitHub Actions. CI workflow must build for ARM64 and AMD64, run tests, and be maintained as part of the project.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Continuous Mount Health Monitoring (Priority: P1)

As a media server operator, I need the system to continuously monitor the health of my debrid WebDAV mounts so that I am immediately aware when mounts become unavailable or unresponsive.

**Why this priority**: Without health monitoring, there is no foundation for any other functionality. This is the core sensing capability that everything else depends on.

**Independent Test**: Can be fully tested by configuring a mount path, starting the monitor, and verifying it detects both healthy and unhealthy mount states. Delivers value by providing visibility into mount health status.

**Acceptance Scenarios**:

1. **Given** a configured mount path exists and is accessible, **When** the monitor starts, **Then** it reports the mount as healthy within the configured check interval.
2. **Given** a configured mount path becomes inaccessible (unmounted, network failure, permission denied), **When** the next health check runs, **Then** the monitor detects and reports the unhealthy state.
3. **Given** a mount was previously unhealthy, **When** it becomes accessible again, **Then** the monitor detects and reports the recovery to healthy state.
4. **Given** the monitor is running, **When** a health check completes, **Then** the result is logged with timestamp and mount path.

---

### User Story 2 - Pod Restart via Health Check Failure (Priority: P2)

As a media server operator, I need the pod to be automatically restarted by Kubernetes when mounts become unhealthy, so that services don't operate against failed mounts and corrupt their metadata.

**Why this priority**: This is the primary protective action that prevents data corruption. It depends on P1 (health monitoring) being functional.

**Independent Test**: Can be fully tested by simulating a mount failure and verifying the readiness probe returns failure status. Kubernetes can be configured to restart pods based on prolonged readiness failures. Delivers value by automatically protecting media servers from metadata corruption.

**Acceptance Scenarios**:

1. **Given** a mount is healthy, **When** the Kubernetes readiness probe queries the health endpoint, **Then** the endpoint returns success (HTTP 200).
2. **Given** a mount becomes unhealthy and remains unhealthy past the debounce threshold, **When** the Kubernetes readiness probe queries the health endpoint, **Then** the endpoint returns failure (HTTP 503).
3. **Given** a mount becomes unhealthy, **When** the unhealthy state is transient (recovers within debounce period), **Then** the readiness endpoint continues returning success (false positive protection).
4. **Given** the readiness endpoint returns failure, **When** Kubernetes restarts the pod (via configured restart policy), **Then** the restart is logged before shutdown.
5. **Given** the service is running, **When** the liveness probe is queried, **Then** the endpoint returns success (HTTP 200) indicating the process is alive.

---

### User Story 3 - Service Startup Gating (Priority: P3)

As a media server operator, I need dependent services to be prevented from starting until mounts are verified healthy, so that services don't initialize against empty or failed mount points.

**Why this priority**: This prevents corruption during service startup. It's less critical than runtime protection (P2) because startup is less frequent than ongoing operation.

**Independent Test**: Can be fully tested by starting the monitor with unhealthy mounts and verifying it blocks or signals dependent services to wait. Delivers value by preventing startup-time metadata corruption.

**Acceptance Scenarios**:

1. **Given** mounts are unhealthy at startup, **When** a dependent service queries readiness, **Then** the monitor indicates not ready.
2. **Given** mounts become healthy after initial unhealthy state, **When** a dependent service queries readiness, **Then** the monitor indicates ready.
3. **Given** the monitor is providing a health endpoint, **When** mounts are healthy, **Then** the endpoint returns a success status.
4. **Given** the monitor is providing a health endpoint, **When** any configured mount is unhealthy, **Then** the endpoint returns a failure status.

---

### User Story 4 - Graceful Shutdown (Priority: P4)

As a container orchestrator (Kubernetes, Docker), I need the monitor to shut down gracefully when receiving termination signals, so that in-flight operations complete cleanly and no resources are leaked.

**Why this priority**: Required for production deployment but doesn't affect core functionality. Enables zero-downtime deployments.

**Independent Test**: Can be fully tested by sending SIGTERM to a running monitor and verifying it exits cleanly within timeout.

**Acceptance Scenarios**:

1. **Given** the monitor is running, **When** SIGTERM is received, **Then** it initiates graceful shutdown and exits with code 0.
2. **Given** the monitor is running, **When** SIGINT is received, **Then** it initiates graceful shutdown and exits with code 0.
3. **Given** a health check is in progress, **When** shutdown is initiated, **Then** the in-flight check is allowed to complete or is cleanly cancelled.
4. **Given** shutdown is initiated, **When** shutdown completes, **Then** all resources are released and final status is logged.

---

### Edge Cases

- What happens when the mount path doesn't exist at startup? System logs error and treats as unhealthy; readiness probe fails.
- What happens when the monitor loses permission to access the mount mid-operation? Detected as unhealthy on next check.
- What happens when multiple mounts are configured and only some fail? Each mount is tracked independently; any unhealthy mount causes probe failure.
- What happens when the monitor itself crashes? Kubernetes restarts the pod; no persistent state required.
- What happens during very brief network glitches? Debounce/threshold prevents false positive probe failures (liveness only).
- What happens if Kubernetes probes faster than the monitor's check interval? Probe returns last known state; does not trigger new filesystem check.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST monitor one or more configured mount paths for health status.
- **FR-002**: System MUST detect mount failures by reading a configurable canary file within each mount path, with a configurable timeout (default: 5 seconds) to detect stale/hung mounts. Detectable failures include: path not found, permission denied, I/O errors, read timeout (stale mount).
- **FR-003**: System MUST perform health checks at a configurable interval (default: 30 seconds).
- **FR-004**: System MUST implement debounce/threshold logic to prevent false positive restarts from transient failures (default: 3 consecutive failures before action).
- **FR-005**: System MUST expose an HTTP liveness endpoint (/healthz/live) that returns HTTP 200 when the service process is running. This indicates the service is alive and able to respond to requests.
- **FR-006**: System MUST expose an HTTP readiness endpoint (/healthz/ready) that returns HTTP 503 when any mount is unhealthy (past debounce threshold), preventing traffic routing until healthy. Kubernetes can be configured to restart pods based on prolonged readiness failures.
- **FR-007**: System MUST log all health state transitions with timestamp, mount path, and previous/new state.
- **FR-008**: System MUST log all health endpoint responses (probe queries) with timestamp and result.
- **FR-009**: System MUST handle SIGTERM and SIGINT signals for graceful shutdown.
- **FR-010**: System MUST complete graceful shutdown within 30 seconds.
- **FR-011**: System MUST accept all configuration via environment variables or command-line flags.
- **FR-012**: System MUST output logs to stdout (info/debug) and stderr (errors/warnings).
- **FR-013**: System MUST support structured (JSON) log format.
- **FR-014**: System MUST exit with code 0 on successful shutdown and non-zero on errors.
- **FR-015**: System MUST support separate endpoints for liveness (/healthz/live) and readiness (/healthz/ready) probes.
- **FR-016**: Project MUST include GitHub Actions CI workflow that builds for both ARM64 and AMD64 architectures and runs all tests.

### Key Entities

- **Mount**: A filesystem path to monitor. Attributes: path, canary file path, health status (healthy/unhealthy), last check time, consecutive failure count, debounce state, last error message.
- **Health Check Result**: The outcome of a single health check. Attributes: mount path, timestamp, status, error message (if any).
- **Health State Transition**: A change in mount health status. Attributes: mount path, timestamp, previous state, new state, trigger (check result or recovery).
- **Probe Response**: The response to a Kubernetes probe request. Attributes: probe type (liveness/readiness), timestamp, HTTP status code, aggregate mount health.

## Assumptions

- The monitor runs as a sidecar container within a Kubernetes pod alongside the media server container.
- The monitor runs with sufficient filesystem permissions to access configured mount paths (shared volume mounts).
- Kubernetes is configured with liveness and readiness probes pointing to the monitor's health endpoints.
- Kubernetes will restart the pod when liveness probe fails (standard Kubernetes behavior).
- A canary file exists at a known path within each mount (e.g., `.health-check` or any stable file). The operator configures this path per mount.
- Network-based health checks (WebDAV HTTP requests) are not required; local mount accessibility via canary file read is sufficient.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Mount failures are detected within 2x the configured check interval (default: 60 seconds).
- **SC-002**: False positive restart rate is less than 1% (transient failures don't trigger restarts).
- **SC-003**: Liveness probe returns failure status within 5 seconds of confirmed unhealthy state (debounce threshold crossed).
- **SC-004**: Graceful shutdown completes within 30 seconds of receiving termination signal.
- **SC-005**: Health endpoint responds within 100 milliseconds.
- **SC-006**: System starts and begins monitoring within 5 seconds of launch.
- **SC-007**: Memory usage remains stable over extended operation (no leaks over 24+ hours).
- **SC-008**: All health state transitions and restart actions are logged with complete context.
