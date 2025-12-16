# Feature Specification: JSON Configuration File

**Feature Branch**: `002-json-config`
**Created**: 2025-12-14
**Status**: Implemented ✅
**Input**: User description: "create a feature to modify the configuration injection. configuration should be injected via config file. The format of the configuration file should be JSON. Useful attributes are: mount name, path to canary file, check frequency, failure threshold."

## Clarifications

### Session 2025-12-14

- Q: Should the system check a default config file location when `--config` is not specified? → A: Yes, check `./config.json` in current working directory as default
- Q: What should be logged when the configuration file is loaded at startup? → A: Verbose - log config file path, all mount paths, and all settings at info level

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Load Configuration from JSON File (Priority: P1)

As a system administrator, I want to configure the mount health monitor using a JSON configuration file so that I can manage complex multi-mount setups with per-mount settings in a single, version-controlled file.

**Why this priority**: This is the core functionality that enables configuration file injection. Without this, the feature provides no value.

**Independent Test**: Can be fully tested by creating a JSON config file with mount definitions and starting the application with the config file path. Delivers the primary value of file-based configuration.

**Acceptance Scenarios**:

1. **Given** a valid JSON configuration file exists at a specified path, **When** the application starts with the config file path specified, **Then** the application loads all mount configurations from the file and begins monitoring
2. **Given** a JSON configuration file with multiple mount definitions, **When** the application loads the configuration, **Then** each mount is configured with its individual settings (canary file, failure threshold)
3. **Given** no configuration file is specified, **When** the application starts, **Then** the application falls back to environment variables and CLI flags (existing behavior preserved)

---

### User Story 2 - Per-Mount Configuration (Priority: P2)

As a system administrator managing heterogeneous storage mounts, I want each mount to have its own canary file path and failure threshold so that I can tune monitoring sensitivity per mount based on its characteristics.

**Why this priority**: This extends the core file loading to support per-mount customization, which is the main advantage over the current global configuration.

**Independent Test**: Can be tested by creating a config file with two mounts having different canary files and thresholds, then verifying each mount uses its specific settings.

**Acceptance Scenarios**:

1. **Given** a mount configuration specifies a custom canary file, **When** health checks run, **Then** the system checks for that specific canary file path within the mount
2. **Given** a mount configuration specifies a failure threshold of 5, **When** 4 consecutive check failures occur for that mount, **Then** the mount remains in degraded state (not unhealthy)
3. **Given** a mount does not specify a canary file, **When** the configuration loads, **Then** the system uses the global default canary file name

---

### User Story 3 - Configuration Validation on Startup (Priority: P3)

As a system administrator, I want the application to validate my configuration file at startup so that I am immediately notified of configuration errors rather than experiencing unexpected runtime behavior.

**Why this priority**: Validation prevents subtle runtime issues but is secondary to the core loading functionality.

**Independent Test**: Can be tested by providing malformed or invalid configuration files and verifying appropriate error messages are returned.

**Acceptance Scenarios**:

1. **Given** a JSON configuration file with invalid JSON syntax, **When** the application attempts to load it, **Then** the application fails to start and displays a clear error message indicating the parse error
2. **Given** a JSON configuration file with a mount missing the required path field, **When** the application attempts to load it, **Then** the application fails to start with a validation error specifying which mount is misconfigured
3. **Given** a JSON configuration file with an invalid failure threshold (negative value), **When** the application attempts to load it, **Then** the application fails to start with a validation error (note: zero means "use global default")

---

### Edge Cases

- What happens when the configuration file path is specified but the file does not exist? Application fails to start with a clear "file not found" error message.
- What happens when the configuration file is empty (valid JSON but no content)? Application fails validation requiring at least one mount to be defined.
- What happens when a mount path in the configuration does not exist on the filesystem? Application logs a warning but starts; the mount will show as unhealthy on first check.
- How does the system handle configuration changes while running? Configuration is loaded at startup only; changes require restart (hot-reload is out of scope).
- What happens when both a config file and environment variables define mount paths? Environment variables override the configuration file (maintaining the existing precedence: Defaults → Config File → Env Vars → CLI Flags).
- What happens when `./config.json` exists but `--config` points to a different file? The explicitly specified `--config` path takes precedence; the default `./config.json` is only used when no `--config` flag is provided.
- What happens when neither `--config` is specified nor `./config.json` exists? Application proceeds using environment variables and CLI flags only (no error, backwards compatible).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST accept a configuration file path via command-line flag (`--config` or `-c`), and MUST check for `./config.json` in the current working directory as a default when no flag is provided
- **FR-002**: System MUST parse JSON configuration files and extract mount definitions
- **FR-003**: System MUST support per-mount configuration including: mount name, mount path, canary file path, and failure threshold
- **FR-004**: System MUST validate configuration file structure and content at startup before beginning monitoring
- **FR-005**: System MUST display clear, actionable error messages when configuration validation fails
- **FR-006**: System MUST apply default values when optional mount settings are not specified (canary file defaults to global setting, failure threshold defaults to global setting)
- **FR-007**: System MUST preserve existing configuration precedence: defaults are overridden by config file, which is overridden by environment variables, which is overridden by CLI flags
- **FR-008**: System MUST include mount name in health status responses and logs when a name is configured
- **FR-009**: System MUST continue to function without a configuration file (backwards compatible with existing env var/CLI flag configuration)
- **FR-010**: System MUST log at info level on startup: the config file path used (or "default" / "none"), all configured mount paths, and their individual settings (canary file, failure threshold)

### Key Entities

- **Mount Configuration**: Represents a single mount's monitoring settings
  - Name: Human-readable identifier for the mount (optional, for logging/status display)
  - Path: Filesystem path to the mount point (required)
  - Canary File: Relative path to the health check file within the mount (optional, inherits global default)
  - Failure Threshold: Number of consecutive failures before marking unhealthy (optional, inherits global default)

- **Application Configuration**: Top-level configuration structure
  - Config File Path: Path to JSON configuration file (optional, new CLI flag)
  - Check Interval: How frequently to perform health checks (global setting)
  - Mounts: Array of mount configurations

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Administrators can configure a multi-mount setup with 10+ mounts using a single configuration file
- **SC-002**: Application startup with a valid configuration file completes within 5 seconds (configuration loading < 100ms)
- **SC-003**: Configuration errors are detected and reported at startup, preventing runtime failures due to misconfiguration
- **SC-004**: Administrators can identify which mount experienced an issue by its configured name in logs and health status responses
- **SC-005**: Existing deployments using only environment variables or CLI flags continue to work without modification (100% backwards compatibility)
