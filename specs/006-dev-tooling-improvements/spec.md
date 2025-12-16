# Feature Specification: Development Tooling Improvements

**Feature Branch**: `006-dev-tooling-improvements`
**Created**: 2025-12-16
**Status**: Draft
**Input**: User description: "Implement issue #21, #24, #28, #25"

## Clarifications

### Session 2025-12-16

*No clarifications needed yet - requirements are well-defined in referenced GitHub issues.*

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Automated KIND Validation (Priority: P1)

As a developer, I need a single `make kind-test` command that creates a KIND cluster, deploys the watchdog, simulates a mount failure, verifies pod restart behavior, and cleans up, so that I can validate end-to-end watchdog functionality without manual intervention.

**Why this priority**: This is the primary deliverable that enables continuous validation of the watchdog feature. Without automated testing, regression detection relies on manual effort which is error-prone and time-consuming.

**Independent Test**: Can be fully tested by running `make kind-test` from a clean state (no existing cluster) and verifying it completes successfully with a pass/fail result. Delivers immediate value by providing automated regression testing.

**Acceptance Scenarios**:

1. **Given** Docker is running and no KIND cluster exists, **When** the developer runs `make kind-test`, **Then** a temporary KIND cluster is created, the monitor is deployed with watchdog enabled, a mount failure is simulated, pod restart is verified, and the cluster is deleted.

2. **Given** Docker is running and a KIND cluster already exists, **When** the developer runs `make kind-test`, **Then** the existing cluster is reused (or recreated with a unique name) and the test proceeds without conflict.

3. **Given** the test completes (pass or fail), **When** cleanup runs, **Then** all test resources are removed and the KIND cluster is deleted (unless `KEEP_CLUSTER=1` is set for debugging).

4. **Given** the watchdog triggers a pod restart during the test, **When** the test script checks the result, **Then** it verifies all containers restarted (not just the monitor) and a WatchdogRestart event was created.

---

### User Story 2 - Comprehensive Watchdog Unit Tests (Priority: P2)

As a developer maintaining the watchdog code, I need comprehensive unit test coverage for all watchdog state transitions, error handling, and edge cases, so that I can refactor with confidence and catch regressions early.

**Why this priority**: Unit tests provide fast feedback during development and are essential for safe refactoring. This builds on the existing test foundation to achieve thorough coverage.

**Independent Test**: Can be tested by running `go test ./tests/unit/... -v` and verifying all new test cases pass. Delivers value by documenting expected behavior and catching regressions.

**Acceptance Scenarios**:

1. **Given** the watchdog package, **When** unit tests are run, **Then** all state transitions (Disabled -> Armed -> PendingRestart -> Triggered) are tested with assertions on state values and logged messages.

2. **Given** a mock K8s client that returns errors, **When** DeletePod fails, **Then** the retry logic with exponential backoff is exercised and verified (correct retry count, backoff timing).

3. **Given** the watchdog is in PendingRestart state, **When** the mount recovers before the restart delay expires, **Then** the pending restart is cancelled and state returns to Armed.

4. **Given** running outside Kubernetes (IsInCluster returns false), **When** watchdog is enabled, **Then** it gracefully degrades to logging-only mode without attempting pod deletion.

5. **Given** RBAC permissions are missing, **When** CanDeletePods returns false, **Then** watchdog logs an error and disables itself.

---

### User Story 3 - Troubleshooting Runbook (Priority: P3)

As an operator deploying the mount monitor, I need a troubleshooting guide that documents common issues, their symptoms, and resolution steps, so that I can diagnose and resolve problems without requiring developer assistance.

**Why this priority**: Documentation reduces support burden and enables self-service debugging. This is important but less urgent than automated testing.

**Independent Test**: Can be verified by reviewing the documentation and confirming it covers the listed issues with clear symptoms and resolutions.

**Acceptance Scenarios**:

1. **Given** a new operator experiencing watchdog issues, **When** they consult the troubleshooting guide, **Then** they find documented symptoms for: RBAC permission errors, missing POD_NAME/POD_NAMESPACE, mount never detected as unhealthy, and pod not restarting.

2. **Given** each documented issue, **When** the operator follows the resolution steps, **Then** they can verify the fix using the provided diagnostic commands (kubectl commands, log queries).

3. **Given** the troubleshooting documentation, **When** a developer reviews it, **Then** it accurately reflects current behavior and configuration options.

---

### User Story 4 - KIND Namespace Customization (Priority: P4)

As a developer running multiple test environments, I need to customize the Kubernetes namespace used by KIND deployments via an environment variable, so that I can run parallel tests without resource conflicts.

**Why this priority**: Namespace customization is a convenience feature that improves developer experience but is not critical for basic functionality.

**Independent Test**: Can be tested by setting `KIND_NAMESPACE=custom-ns` and running `make kind-deploy`, then verifying resources are created in the custom namespace.

**Acceptance Scenarios**:

1. **Given** `KIND_NAMESPACE` environment variable is set to "test-ns", **When** `make kind-deploy` runs, **Then** all resources (deployment, configmap, RBAC) are created in the "test-ns" namespace.

2. **Given** `KIND_NAMESPACE` is not set, **When** `make kind-deploy` runs, **Then** resources are created in the default namespace (currently "default").

3. **Given** a custom namespace is specified, **When** the namespace doesn't exist, **Then** the deployment script creates it automatically.

---

### Edge Cases

- What happens when `make kind-test` is interrupted mid-execution? The test should be idempotent; re-running cleans up partial state.
- What happens when Docker daemon is not running? Clear error message indicating Docker must be running.
- What happens when `kind` binary is not installed? Error message with installation instructions.
- What happens when the canary file can't be deleted (permission denied)? Test fails with clear error indicating permission issue.
- How does the automated test handle timing variations? Use polling with timeout rather than fixed sleep durations.

## Requirements *(mandatory)*

### Functional Requirements

**Automated KIND Testing (US1)**

- **FR-001**: Project MUST provide a `make kind-test` target that runs the full end-to-end watchdog test cycle.
- **FR-002**: The kind-test target MUST create a KIND cluster if one doesn't exist, or reuse an existing one.
- **FR-003**: The kind-test target MUST deploy the monitor with watchdog enabled and a short restart delay (e.g., 5s) for fast testing.
- **FR-004**: The kind-test target MUST simulate mount failure by removing the canary file via `kubectl exec`.
- **FR-005**: The kind-test target MUST verify pod restart occurred by checking pod creation timestamps or restart counts.
- **FR-006**: The kind-test target MUST verify a Kubernetes event with reason "WatchdogRestart" was created.
- **FR-007**: The kind-test target MUST clean up all test resources and delete the cluster on completion (configurable via KEEP_CLUSTER).
- **FR-008**: The kind-test target MUST exit with code 0 on success and non-zero on failure.
- **FR-009**: The kind-test target MUST complete within 5 minutes under normal conditions.

**Watchdog Unit Tests (US2)**

- **FR-010**: Unit tests MUST cover all watchdog state transitions with explicit assertions.
- **FR-011**: Unit tests MUST cover DeletePod retry logic with mock failures.
- **FR-012**: Unit tests MUST cover restart cancellation when mount recovers.
- **FR-013**: Unit tests MUST cover graceful degradation outside Kubernetes.
- **FR-014**: Unit tests MUST cover RBAC validation failure handling.
- **FR-015**: Unit tests MUST achieve at least 80% code coverage for the watchdog package.

**Troubleshooting Runbook (US3)**

- **FR-016**: Project MUST provide troubleshooting documentation in `docs/troubleshooting.md` or equivalent location.
- **FR-017**: Troubleshooting documentation MUST include sections for: RBAC issues, environment variable configuration, mount detection, and pod restart failures.
- **FR-018**: Each troubleshooting section MUST include: symptom description, diagnostic commands, and resolution steps.

**Namespace Customization (US4)**

- **FR-019**: KIND deployment scripts MUST support a `KIND_NAMESPACE` environment variable to customize the target namespace.
- **FR-020**: When `KIND_NAMESPACE` is set, all Kubernetes resources MUST be created in that namespace.
- **FR-021**: The deployment script MUST create the namespace if it doesn't exist.

### Key Entities

- **Test Script**: Shell script or Makefile target that orchestrates the end-to-end test. Attributes: cluster name, namespace, timeout values, cleanup behavior.
- **Mock K8sClient**: Test double implementing the K8sClient interface for unit testing. Attributes: configurable responses, call recording for assertions.
- **Troubleshooting Entry**: Documentation entry for a specific issue. Attributes: symptom, diagnostic commands, resolution steps, related configuration.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `make kind-test` completes successfully on a clean machine with Docker installed within 5 minutes.
- **SC-002**: `make kind-test` detects watchdog failures (e.g., pod not restarting) and exits with non-zero code.
- **SC-003**: Watchdog package unit test coverage reaches at least 80% as measured by `go test -cover`.
- **SC-004**: All documented troubleshooting scenarios include working diagnostic commands that can be copy-pasted.
- **SC-005**: Setting `KIND_NAMESPACE=custom` results in all resources being created in the "custom" namespace.
- **SC-006**: CI pipeline can run `make kind-test` as part of the test suite (if Docker-in-Docker or similar is available).

## Related Specifications

- **Extends**: [005-pod-restart-watchdog](../005-pod-restart-watchdog/spec.md) - Adds automated testing and documentation for watchdog functionality
- **Uses**: [004-kind-local-dev](../004-kind-local-dev/spec.md) - Builds on existing KIND infrastructure

## Referenced Issues

- **#21**: Add make target for running automated end-to-end watchdog tests in KIND
- **#24**: Add comprehensive unit test coverage for watchdog package
- **#25**: Add namespace customization support for KIND deployments
- **#28**: Watchdog unit test coverage improvements
