# Feature Specification: Pod Restart Watchdog

**Feature Branch**: `005-pod-restart-watchdog`
**Created**: 2025-12-15
**Status**: Draft
**Input**: User description: "I need this service to act as a watchdog sidecar. When the checks fail, it needs to force a restart on the entire pod, rather than just a single container"

## Clarifications

### Session 2025-12-15

- Q: How should the watchdog handle restart loops (mount keeps failing after restart)? → A: No protection - restart immediately each time, trust Kubernetes CrashLoopBackOff mechanism.
- Q: What happens after all pod deletion retries are exhausted? → A: Fall back to process exit (non-zero) to trigger container restart via liveness probe.
- Q: Should watchdog mode be enabled or disabled by default? → A: Disabled by default - operators must explicitly enable for safe rollout.
- Q: What should the default restart delay be after mount becomes UNHEALTHY? → A: 0 seconds (immediate) - existing debounce threshold already filters transient failures.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Mount Failure Triggers Pod Restart (Priority: P1)

As an operator, when a mount becomes unhealthy and stays unhealthy beyond the configured threshold, I need the watchdog to force a restart of the entire pod so that all containers (including my main application) are restarted together with fresh mount connections.

**Why this priority**: This is the core functionality requested. Without pod-level restarts, only the monitor container restarts while the main application continues running with a potentially broken mount, leading to application errors and data corruption.

**Independent Test**: Can be fully tested by deploying the watchdog sidecar, simulating a mount failure (removing the canary file), and verifying that the entire pod restarts (all containers terminate and new pod is created).

**Acceptance Scenarios**:

1. **Given** a pod with watchdog mode enabled and healthy mounts, **When** the mount becomes unhealthy and remains unhealthy for the configured failure threshold, **Then** the watchdog deletes the pod via the Kubernetes API and the deployment controller creates a replacement pod.

2. **Given** a pod with watchdog mode enabled, **When** the mount becomes unhealthy but recovers before the failure threshold is exceeded, **Then** the watchdog does NOT trigger a pod restart.

3. **Given** a pod with watchdog mode enabled and multiple mounts, **When** any single mount becomes unhealthy beyond the threshold, **Then** the watchdog triggers a pod restart.

---

### User Story 2 - Configurable Watchdog Behavior (Priority: P2)

As an operator, I need to configure whether watchdog mode is enabled or disabled, and customize the restart behavior, so that I can adapt the service to different deployment scenarios (development vs production).

**Why this priority**: Configuration flexibility is essential for safe rollout. Operators need to test in non-watchdog mode first, then enable watchdog mode when confident. Some environments may prefer the current container-only restart behavior.

**Independent Test**: Can be tested by deploying with watchdog mode disabled, simulating mount failure, and verifying that pod-level restart does NOT occur (only readiness probe fails).

**Acceptance Scenarios**:

1. **Given** watchdog mode is disabled in configuration, **When** mount health checks fail, **Then** the service behaves as before (HTTP 503 on probes) without triggering pod deletion.

2. **Given** watchdog mode is enabled with a custom restart delay, **When** mount becomes unhealthy, **Then** the watchdog waits for the configured delay before triggering pod deletion.

3. **Given** watchdog mode is enabled, **When** the operator changes the configuration at runtime (if supported), **Then** the new configuration takes effect without requiring pod restart. *(Note: Runtime config reload is out of scope for MVP; requires pod restart to apply config changes.)*

---

### User Story 3 - Restart Event Visibility (Priority: P3)

As an operator, I need visibility into watchdog-triggered restarts through logs and Kubernetes events so that I can monitor the system health, debug issues, and set up alerts.

**Why this priority**: Observability is critical for production operations but can be added after core functionality works. Operators need to distinguish watchdog restarts from other pod restarts.

**Independent Test**: Can be tested by triggering a watchdog restart and verifying that structured logs and Kubernetes events are created with appropriate details.

**Acceptance Scenarios**:

1. **Given** watchdog mode is enabled, **When** the watchdog triggers a pod restart, **Then** a structured log entry is created with mount path, failure reason, and restart timestamp.

2. **Given** watchdog mode is enabled, **When** the watchdog triggers a pod restart, **Then** a Kubernetes event is created on the pod with reason "WatchdogRestart" and a descriptive message.

3. **Given** watchdog mode is enabled, **When** the watchdog attempts to restart but fails (API error), **Then** an error is logged with details and the watchdog retries with exponential backoff.

---

### Edge Cases

- What happens when the Kubernetes API is temporarily unavailable?
  - The watchdog should retry with exponential backoff and log errors. After 3 failed retries, the watchdog exits with non-zero code to trigger container restart via liveness probe as a fallback mechanism.

- What happens when the pod is already being terminated?
  - The watchdog should detect termination-in-progress and skip deletion.

- What happens when the watchdog lacks RBAC permissions to delete pods?
  - The watchdog should log a clear error message at startup indicating missing permissions.

- What happens when running outside Kubernetes (local development)?
  - The watchdog mode should gracefully degrade to logging-only (no actual restart).

- What happens if the mount recovers while the restart delay is counting down?
  - The pending restart should be cancelled.

- What happens if mounts keep failing after pod restart (restart loop)?
  - The watchdog does not implement its own backoff; it relies on Kubernetes CrashLoopBackOff to rate-limit pod restarts. The watchdog will trigger deletion immediately upon reaching the unhealthy threshold each time.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a configuration option to enable or disable watchdog mode. Watchdog mode MUST be disabled by default for safe rollout.
- **FR-002**: System MUST use the Kubernetes API with in-cluster authentication to delete its own pod when watchdog mode is triggered.
- **FR-003**: System MUST delete the pod only after a mount has been in UNHEALTHY state for a configurable minimum duration (restart delay). Default: 0 seconds (immediate), since the existing debounce threshold already filters transient failures.
- **FR-004**: System MUST cancel pending restarts if the mount recovers to HEALTHY before the restart delay expires.
- **FR-005**: System MUST emit structured log entries for all watchdog-related state changes (armed, triggered, cancelled, failed).
- **FR-006**: System MUST create a Kubernetes event on the pod when triggering a restart, including the reason and affected mount path.
- **FR-007**: System MUST validate RBAC permissions at startup and log a clear error if pod delete permission is missing.
- **FR-008**: System MUST gracefully handle running outside Kubernetes by disabling pod deletion and logging a warning.
- **FR-009**: System MUST retry pod deletion on transient API failures with exponential backoff (max 3 retries). After retries exhausted, system MUST exit with non-zero code to trigger container restart via liveness probe as fallback.
- **FR-010**: System MUST detect if the pod is already terminating and skip redundant deletion requests.
- **FR-011**: System MUST require appropriate Kubernetes RBAC resources (ServiceAccount, Role, RoleBinding) to be deployed.
- **FR-012**: System MUST include RBAC manifests as part of the deployment artifacts.

### Key Entities

- **WatchdogState**: Represents the current state of the watchdog (disabled, armed, pending_restart, triggered). Tracks time since first failure and restart delay countdown.
- **RestartEvent**: Represents a pod restart event with timestamp, mount path, failure count, and reason.
- **K8sClient**: Abstraction for interacting with the Kubernetes API (pod deletion, event creation).

### Assumptions

- The Kubernetes cluster supports RBAC (Kubernetes 1.6+).
- The deployment uses a Deployment or StatefulSet controller that will recreate deleted pods.
- In-cluster Kubernetes configuration is available at the standard paths (`/var/run/secrets/kubernetes.io/serviceaccount/`).
- The pod name and namespace are available via the Downward API (environment variables or volume mounts).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: When a mount is unhealthy for longer than the restart delay, the pod is deleted and replaced within 60 seconds of the threshold being exceeded.
- **SC-002**: When watchdog triggers a restart, all containers in the pod terminate together (verified by matching termination timestamps within 5 seconds).
- **SC-003**: Watchdog restart events are visible in `kubectl get events` output within 10 seconds of the restart trigger.
- **SC-004**: Operators can enable/disable watchdog mode without code changes, using only configuration file or environment variable.
- **SC-005**: 100% of watchdog restarts are logged with structured fields (mount_path, reason, timestamp) for alerting integration.
- **SC-006**: The watchdog correctly handles API failures, retrying at least 3 times before giving up.
