# Feature Specification: KIND Local Development Environment

**Feature Branch**: `004-kind-local-dev`
**Created**: 2025-12-15
**Status**: Draft
**Input**: User description: "add kind (kubernetes in docker) for local development and testing"

## Clarifications

### Session 2025-12-15

- Q: How should developers simulate mount failures in the KIND environment? → A: Users simulate failures manually (e.g., kubectl exec to remove/restore canary files). No automated Makefile targets required.
- Q: What naming convention should be used for KIND clusters? → A: User-customizable via environment variable (e.g., KIND_CLUSTER_NAME) with a default value.
- Q: How should the monitor be configured in KIND? → A: Use a JSON config file mounted as a ConfigMap rather than environment variables. This exercises the JSON config parsing code path and demonstrates per-mount configuration.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Local KIND Cluster Setup (Priority: P1)

As a developer, I need to quickly spin up a local Kubernetes cluster using KIND so that I can test the mount-monitor sidecar in a real Kubernetes environment without requiring access to a remote cluster.

**Why this priority**: This is the foundational capability. Without a local cluster, no other local Kubernetes testing is possible. It enables the entire local development workflow.

**Independent Test**: Can be fully tested by running a single command to create a KIND cluster and verifying it's accessible via kubectl. Delivers immediate value by providing a local Kubernetes environment.

**Acceptance Scenarios**:

1. **Given** Docker is installed and running, **When** the developer runs the cluster creation command, **Then** a KIND cluster is created within 2 minutes.
2. **Given** a KIND cluster exists, **When** the developer runs kubectl commands, **Then** they can interact with the cluster (list nodes, namespaces, etc.).
3. **Given** a KIND cluster exists, **When** the developer runs the cluster deletion command, **Then** the cluster is completely removed and resources are freed.
4. **Given** Docker is not running, **When** the developer attempts to create a cluster, **Then** a clear error message indicates Docker must be running.

---

### User Story 2 - Deploy Monitor to Local Cluster (Priority: P2)

As a developer, I need to deploy the mount-monitor service to my local KIND cluster so that I can test its behavior with Kubernetes probes, pod lifecycle, and sidecar patterns.

**Why this priority**: Deploying the monitor is the primary use case for the local cluster. It depends on P1 (cluster creation) and enables testing the monitor's Kubernetes integration.

**Independent Test**: Can be fully tested by deploying the monitor and verifying the pods are running with correct probe configuration. Delivers value by enabling Kubernetes-specific testing.

**Acceptance Scenarios**:

1. **Given** a KIND cluster is running and the monitor image is built, **When** the developer runs the deployment command, **Then** the monitor is deployed as a sidecar alongside a test application.
2. **Given** the monitor is deployed, **When** the developer checks pod status, **Then** both containers (main app and sidecar) are running.
3. **Given** the monitor is deployed, **When** the developer queries the health endpoints from within the cluster, **Then** the endpoints respond correctly (liveness: 200, readiness: 200 when healthy).
4. **Given** the monitor is deployed, **When** the developer views logs, **Then** the monitor's structured JSON logs are visible showing health check activity.

---

### User Story 3 - Simulate Mount Failures (Priority: P3)

As a developer, I need to simulate mount health and failure scenarios so that I can verify the monitor correctly detects failures and triggers pod restarts via probe failures.

**Why this priority**: This enables testing the core value proposition of the monitor - detecting failures and triggering restarts. It depends on P2 (monitor deployed) to be meaningful.

**Independent Test**: Can be fully tested by manipulating the simulated mount state and observing the monitor's response. Delivers value by validating the monitor's failure detection logic in a real Kubernetes environment.

**Acceptance Scenarios**:

1. **Given** the monitor is deployed and mounts are healthy, **When** the developer simulates a mount failure (removes canary file), **Then** the monitor detects the failure within the check interval.
2. **Given** a mount failure is detected and persists past the debounce threshold, **When** Kubernetes queries the readiness probe, **Then** the probe returns failure (503) and pod status changes.
3. **Given** the readiness probe is failing, **When** Kubernetes restart policy is triggered, **Then** the pod is restarted and the restart is visible in events.
4. **Given** a simulated failure exists, **When** the developer restores mount health (recreates canary file), **Then** the monitor detects recovery and readiness probe returns success.

---

### User Story 4 - Quick Iteration Workflow (Priority: P4)

As a developer, I need to quickly rebuild and redeploy changes to the monitor so that I can iterate rapidly during development without manual steps.

**Why this priority**: Improves developer experience but is not essential for basic functionality. Builds on P1-P3 to provide a smooth development workflow.

**Independent Test**: Can be fully tested by making a code change and running a single command to rebuild and redeploy. Delivers value by reducing friction during development.

**Acceptance Scenarios**:

1. **Given** the developer modifies monitor source code, **When** they run the rebuild-and-deploy command, **Then** the new image is built, loaded into KIND, and pods are restarted with the new image.
2. **Given** a rebuild is in progress, **When** the process completes, **Then** the new version is running within 60 seconds of command execution.

---

### Edge Cases

- What happens when Docker runs out of disk space during cluster creation? Clear error message and cleanup guidance should be provided.
- What happens if the KIND cluster is corrupted or in a bad state? A force-delete option should recover without manual intervention.
- What happens if multiple developers run KIND clusters simultaneously? Cluster name is customizable via `KIND_CLUSTER_NAME` env var; developers can set different names to run parallel clusters without conflict.
- What happens if the monitor image fails to load into KIND? Error message indicates the issue and the image can be rebuilt.
- What happens during rapid iteration when pods are still terminating? Deployment waits for clean state or forces replacement.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Project MUST provide a KIND cluster configuration file that defines a single-node cluster suitable for local development.
- **FR-002**: Project MUST provide Kubernetes manifests for deploying the monitor as a sidecar alongside a simple test application (e.g., busybox or alpine). The monitor MUST be configured via a JSON config file mounted as a ConfigMap (not environment variables) to exercise the JSON config parsing code path.
- **FR-003**: Project MUST provide a script to create a local KIND cluster with a single command.
- **FR-004**: Project MUST provide a script to delete the local KIND cluster and clean up all resources.
- **FR-005**: Project MUST provide a script or Makefile target to build the monitor image and load it into the KIND cluster.
- **FR-006**: Project MUST provide a script or Makefile target to deploy the monitor to the KIND cluster.
- **FR-007**: Project MUST deploy with pre-configured healthy mounts (local directories with canary files present).
- **FR-008**: Project MUST document how developers can manually simulate mount failures (e.g., kubectl exec to remove canary files) and recovery (restore canary files).
- **FR-009**: Project MUST configure Kubernetes liveness and readiness probes in the deployment manifests.
- **FR-010**: Project MUST document the complete local development workflow in README or dedicated documentation.
- **FR-011**: Project MUST provide Makefile targets for common operations: `kind-create`, `kind-delete`, `kind-deploy`, `kind-logs`. Additional convenience targets (`kind-status`, `kind-undeploy`, `kind-redeploy`, `kind-help`) SHOULD be provided.

### Key Entities

- **KIND Cluster**: A local Kubernetes cluster running in Docker containers. Attributes: cluster name (customizable via `KIND_CLUSTER_NAME` env var, defaults to project name), Kubernetes version, node configuration.
- **Test Deployment**: A Kubernetes Deployment with the monitor as a sidecar. Attributes: main container (test app), sidecar container (monitor), shared volume for simulated mounts, JSON config file mounted from ConfigMap.
- **Simulated Mount**: A local directory mounted into the pod as a volume to simulate a debrid mount. Attributes: path, canary file presence (healthy/unhealthy).
- **Development Scripts**: Shell scripts or Makefile targets that automate cluster and deployment management.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developer can create a functional KIND cluster in under 2 minutes from a cold start (Docker running, no existing cluster).
- **SC-002**: Developer can deploy the monitor and have it running in under 60 seconds after cluster creation.
- **SC-003**: Developer can simulate a mount failure and observe pod restart behavior within 90 seconds (accounting for debounce threshold and probe intervals).
- **SC-004**: Developer can complete a code-change-to-running-pod iteration cycle in under 60 seconds.
- **SC-005**: All cluster and deployment operations complete without manual intervention (fully automated via scripts/Makefile).
- **SC-006**: Documentation enables a new developer to complete the full workflow on their first attempt.
- **SC-007**: KIND cluster operates correctly on both AMD64 and ARM64 developer machines (Mac M1/M2/M3, Intel, Linux).

## Assumptions

- Docker Desktop or Docker Engine is installed and running on the developer's machine.
- Developer has kubectl installed and available in PATH.
- Developer has KIND installed (or installation is documented/automated).
- Local filesystem directories are sufficient to simulate debrid mounts (no WebDAV required for local testing).
- The KIND environment is for local development only and is not integrated into CI/CD workflows.
- Developer has basic familiarity with Kubernetes concepts (pods, deployments, probes).
