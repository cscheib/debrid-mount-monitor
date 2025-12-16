# Feature Specification: Tech Debt Cleanup

**Feature Branch**: `007-tech-debt-cleanup`
**Created**: 2025-12-16
**Status**: Draft
**Input**: User description: "Tech debt cleanup: Go namespace rename (chris→cscheib), terminology alignment (debounce→failure), path documentation, remove environment variable configuration"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Correct Module Namespace (Priority: P1)

As a developer contributing to this project, I want the Go module namespace to match the actual GitHub repository location so that `go get` and import paths work correctly and don't cause confusion.

**Why this priority**: The module namespace mismatch (`github.com/chris/...` vs actual `github.com/cscheib/...`) is a functional issue that could break builds for external consumers and creates confusion about project ownership.

**Independent Test**: Can be fully tested by running `go build` after the change and verifying all imports resolve correctly with the new namespace.

**Acceptance Scenarios**:

1. **Given** a fresh clone of the repository, **When** a developer runs `go build ./...`, **Then** the build succeeds with no import errors
2. **Given** the updated `go.mod`, **When** examining import statements in all Go files, **Then** all imports use the `github.com/cscheib/debrid-mount-monitor` namespace
3. **Given** the repository, **When** a developer runs `go mod tidy`, **Then** no changes are required (module is already consistent)

---

### User Story 2 - Consistent Failure Threshold Terminology (Priority: P2)

As a user configuring the monitor, I want consistent terminology throughout the configuration and documentation so that I understand what each setting does without confusion between "debounce" and "failure threshold."

**Why this priority**: Inconsistent terminology creates cognitive overhead and potential misconfiguration. The term "failure threshold" is self-documenting, while "debounce" requires understanding signal processing concepts.

**Independent Test**: Can be fully tested by reviewing all configuration options, documentation, and log output to verify consistent use of "failure threshold" terminology.

**Acceptance Scenarios**:

1. **Given** the JSON configuration schema, **When** configuring the global failure threshold, **Then** the key is named `failureThreshold` (matching the per-mount field)
2. **Given** the CLI help output, **When** viewing threshold-related flags, **Then** the flag is named `--failure-threshold` with clear description
3. **Given** a running monitor with a mount that has exceeded the threshold, **When** viewing logs, **Then** log messages use "failure threshold" terminology
4. **Given** the README documentation, **When** reading about threshold configuration, **Then** only "failure threshold" terminology is used (no "debounce" references)

---

### User Story 3 - Clear Path Configuration Documentation (Priority: P3)

As a user configuring mount paths, I want clear documentation about whether paths should be relative or absolute so that I can configure the monitor correctly for my deployment environment.

**Why this priority**: Users may be unsure whether to use relative or absolute paths. Clear documentation prevents misconfiguration and support requests.

**Independent Test**: Can be fully tested by reviewing documentation to verify path format guidance is present and accurate.

**Acceptance Scenarios**:

1. **Given** the README documentation, **When** reading about mount path configuration, **Then** it clearly states that mount paths can be either relative or absolute
2. **Given** the README documentation, **When** reading about canary file configuration, **Then** it clearly states that canary file paths are always relative to the mount path
3. **Given** the JSON configuration example in the README, **When** viewing the mounts section, **Then** examples demonstrate both relative and absolute path usage

---

### User Story 4 - Simplified Configuration (Priority: P4)

As a user deploying the monitor, I want a single, clear configuration method (JSON file) so that I don't have to understand precedence rules between environment variables, CLI flags, and config files.

**Why this priority**: Environment variable configuration adds complexity without significant value. JSON configuration is more expressive (supports mount arrays with per-mount settings) and is the recommended approach. Removing environment variables simplifies the mental model.

**Independent Test**: Can be fully tested by attempting to use environment variables and verifying they have no effect on configuration.

**Acceptance Scenarios**:

1. **Given** the application, **When** setting `MOUNT_PATHS` environment variable, **Then** the variable is ignored (has no effect)
2. **Given** the application, **When** setting `DEBOUNCE_THRESHOLD` environment variable, **Then** the variable is ignored (has no effect)
3. **Given** the README documentation, **When** reading the Configuration section, **Then** no environment variables are documented (only JSON config and CLI flags)
4. **Given** the application startup, **When** no JSON config or CLI flags provided, **Then** appropriate error or default behavior occurs without falling back to environment variables

**Note**: Kubernetes-specific environment variables (`KUBERNETES_SERVICE_HOST`, `KUBERNETES_SERVICE_PORT`, `POD_NAME`, `POD_NAMESPACE`) used by the watchdog feature are NOT affected by this change - they serve a different purpose (runtime environment detection, not user configuration).

---

### Edge Cases

- What happens when a user has existing deployments using environment variables? (Answer: They must migrate to JSON config; this is a breaking change that should be documented in release notes)
- How does the system behave with mixed relative/absolute paths in the same config? (Answer: Each path is resolved independently - relative paths resolve from the working directory)

## Requirements *(mandatory)*

### Functional Requirements

**Namespace Change:**
- **FR-001**: Go module path MUST be `github.com/cscheib/debrid-mount-monitor`
- **FR-002**: All internal package imports MUST use the updated namespace
- **FR-003**: Build MUST succeed with the updated namespace

**Terminology Alignment:**
- **FR-004**: Global configuration key MUST be named `failureThreshold` in JSON config
- **FR-005**: CLI flag MUST be named `--failure-threshold`
- **FR-006**: Internal variable names MUST use "failure" terminology (not "debounce")
- **FR-007**: Log messages MUST use "failure threshold" terminology
- **FR-008**: All code comments MUST use "failure threshold" terminology

**Path Documentation:**
- **FR-009**: Documentation MUST state that mount paths can be relative or absolute
- **FR-010**: Documentation MUST state that canary file paths are relative to mount paths

**Configuration Simplification:**
- **FR-011**: Application MUST NOT read `MOUNT_PATHS` environment variable for configuration
- **FR-012**: Application MUST NOT read `CANARY_FILE` environment variable for configuration
- **FR-013**: Application MUST NOT read `CHECK_INTERVAL` environment variable for configuration
- **FR-014**: Application MUST NOT read `READ_TIMEOUT` environment variable for configuration
- **FR-015**: Application MUST NOT read `SHUTDOWN_TIMEOUT` environment variable for configuration
- **FR-016**: Application MUST NOT read `DEBOUNCE_THRESHOLD` (or `FAILURE_THRESHOLD`) environment variable for configuration
- **FR-017**: Application MUST NOT read `HTTP_PORT` environment variable for configuration
- **FR-018**: Application MUST NOT read `LOG_LEVEL` environment variable for configuration
- **FR-019**: Application MUST NOT read `LOG_FORMAT` environment variable for configuration
- **FR-020**: Application MUST NOT read `WATCHDOG_ENABLED` environment variable for configuration
- **FR-021**: Application MUST NOT read `WATCHDOG_RESTART_DELAY` environment variable for configuration
- **FR-022**: Documentation MUST be updated to remove environment variable references for configuration
- **FR-023**: Kubernetes runtime environment variables (`KUBERNETES_SERVICE_HOST`, `KUBERNETES_SERVICE_PORT`, `POD_NAME`, `POD_NAMESPACE`) MUST continue to be read (not user configuration)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All Go builds succeed with zero import errors after namespace change
- **SC-002**: Zero occurrences of "debounce" terminology in user-facing configuration, documentation, and logs
- **SC-003**: Zero occurrences of configuration-related environment variable reads in application code (excluding Kubernetes runtime variables)
- **SC-004**: Documentation clearly explains path configuration in a single, findable location
- **SC-005**: All existing unit tests pass after changes (with necessary test updates)
- **SC-006**: Breaking change (environment variable removal) is documented for users migrating from previous versions

## Assumptions

- This is a **breaking change** for users relying on environment variable configuration
- The release notes will clearly document the migration path (use JSON config instead)
- The Kubernetes runtime environment variables are distinct from user configuration and remain unchanged
- Relative paths resolve from the current working directory of the process
