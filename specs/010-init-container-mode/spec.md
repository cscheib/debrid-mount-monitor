# Feature Specification: Init Container Mode

**Feature Branch**: `010-init-container-mode`
**Created**: 2025-12-17
**Status**: Draft
**Input**: User description: "new feature: initContainer mode. When in initContainer mode, the service immediately checks for all filesystems defined in the config file. If the check passes, the service will log the success and immediately end with a successful exit code. If the check fails, the service will log the failure and immediately fail with an unsuccessful exit code. The purpose of this feature is to prevent the pod from starting when the filesystems are not available. This mode will be set via a runtime flag --init-container-mode"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Block Pod Startup When Mounts Unavailable (Priority: P1)

As a Kubernetes operator, I want the pod to fail to start if required filesystem mounts are not available, so that my application doesn't run in a broken state with missing data.

**Why this priority**: This is the core value proposition - preventing application pods from starting when their required storage is unavailable. Without this, applications could start and immediately fail or behave unexpectedly.

**Independent Test**: Can be fully tested by running the service with `--init-container-mode` flag against unavailable mounts and verifying exit code 1 with appropriate error logs.

**Acceptance Scenarios**:

1. **Given** one or more mounts are unavailable (canary file cannot be read), **When** the service runs in init-container mode, **Then** the service logs the failure details and exits with a non-zero exit code
2. **Given** one mount is unavailable among several configured mounts, **When** the service runs in init-container mode, **Then** the service identifies the specific failing mount(s) in the log output and exits with a non-zero exit code

---

### User Story 2 - Allow Pod Startup When All Mounts Healthy (Priority: P1)

As a Kubernetes operator, I want the pod to start normally when all required filesystem mounts are available, so that my application can proceed with its work.

**Why this priority**: Equal priority to P1 above as this represents the happy path - the service must correctly identify healthy mounts and exit successfully.

**Independent Test**: Can be fully tested by running the service with `--init-container-mode` flag against available mounts with valid canary files and verifying exit code 0 with success logs.

**Acceptance Scenarios**:

1. **Given** all configured mounts are available and canary files are readable, **When** the service runs in init-container mode, **Then** the service logs success for each mount and exits with exit code 0
2. **Given** a valid configuration file with multiple mounts all healthy, **When** the service runs in init-container mode, **Then** the service completes all checks and exits successfully within a reasonable time

---

### User Story 3 - Provide Clear Diagnostic Output (Priority: P2)

As a Kubernetes operator troubleshooting a failed pod, I want clear log output explaining which mounts failed and why, so that I can quickly diagnose and fix storage issues.

**Why this priority**: While the exit code alone enables the Kubernetes initContainer pattern, diagnostic output is essential for operators to understand and resolve issues efficiently.

**Independent Test**: Can be tested by examining log output when running against both healthy and unhealthy mounts, verifying mount names, paths, and error details are clearly logged.

**Acceptance Scenarios**:

1. **Given** a mount fails due to missing canary file, **When** the service runs in init-container mode, **Then** the log output includes the mount name, path, and specific error reason
2. **Given** all mounts are healthy, **When** the service runs in init-container mode, **Then** the log output confirms each mount was checked successfully with its name and path

---

### Edge Cases

- What happens when the configuration file is missing or invalid? The service should exit with an error code and log the configuration problem.
- What happens when a canary file read times out (slow/hung NFS)? The service should treat the timeout as a failure and exit with a non-zero code.
- What happens when no mounts are configured? The service exits with an error code during configuration validation (existing behavior - at least one mount is required).
- What happens when the mount path exists but the canary file does not? The service should treat this as a mount failure.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Service MUST accept a `--init-container-mode` runtime flag that enables init container behavior
- **FR-002**: When init-container mode is enabled, service MUST check all configured mounts exactly once
- **FR-003**: Service MUST exit with code 0 when all mounts pass health checks
- **FR-004**: Service MUST exit with a non-zero code when any mount fails health checks
- **FR-005**: Service MUST log the health check result for each mount (success or failure with details)
- **FR-006**: Service MUST use the same configuration file format and loading mechanism as the normal service mode
- **FR-007**: Service MUST respect the configured read timeout when checking each mount
- **FR-008**: Service MUST NOT start the HTTP server, monitor loop, or watchdog in init-container mode
- **FR-009**: Service MUST exit after completing checks (not continue running)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Init container mode completes all mount checks and exits within 30 seconds for typical configurations (accounting for read timeouts)
- **SC-002**: Exit code correctly reflects mount health: 0 for all healthy, non-zero for any failure
- **SC-003**: Log output includes mount name and path for every checked mount
- **SC-004**: Failed mount logs include actionable error information (timeout, file not found, permission denied, etc.)

## Assumptions

- The existing `health.Checker` component will be reused for performing mount checks
- The existing `config` package will be reused for loading and validating configuration
- The same canary file strategy used in normal mode will be used in init-container mode
- Structured logging (slog) will be used consistent with the rest of the application
- A single non-zero exit code (e.g., 1) is sufficient for all failure cases; differentiated exit codes are not required
