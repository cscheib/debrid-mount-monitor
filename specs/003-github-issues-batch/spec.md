# Feature Specification: GitHub Issues Batch Implementation

**Feature Branch**: `003-github-issues-batch`
**Created**: 2025-12-15
**Status**: Implemented ✅
**Input**: User description: "Evaluate and implement (or close as will not do) all issues currently in GitHub for this repo"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Security-Conscious Deployment (Priority: P1)

A system administrator deploys the mount monitor in a production environment and wants assurance that the configuration system has basic security hardening against common attack vectors.

**Why this priority**: Security improvements protect production deployments and prevent potential attack vectors like DoS via oversized config files or config file tampering.

**Independent Test**: Can be fully tested by creating malicious config files (oversized, world-writable) and verifying the system handles them appropriately.

**Acceptance Scenarios**:

1. **Given** a config file larger than 1MB, **When** the monitor starts, **Then** it should reject the config with a clear error message indicating the file size limit.
2. **Given** a config file with world-writable permissions (chmod 666), **When** the monitor starts on Unix, **Then** it should log a warning about the security risk but continue loading.

---

### User Story 2 - Operator Debugging Configuration Issues (Priority: P2)

An operator misconfigures the monitor (e.g., readTimeout >= checkInterval) and needs to understand why the configuration is invalid.

**Why this priority**: Clear error messages reduce support burden and help operators self-service configuration issues.

**Independent Test**: Can be tested by providing invalid configurations and verifying error messages explain the constraint violations clearly.

**Acceptance Scenarios**:

1. **Given** a config where readTimeout equals checkInterval, **When** validation runs, **Then** the error message should explain that health checks would overlap or never complete.

---

### User Story 3 - Developer Understanding Code Limitations (Priority: P2)

A developer maintaining the codebase needs to understand the known limitations of the health checker, particularly around goroutine behavior during hung mounts.

**Why this priority**: Documentation prevents developers from wasting time debugging known limitations and helps them make informed decisions about potential improvements.

**Independent Test**: Can be verified by reviewing the code comments in the health checker module.

**Acceptance Scenarios**:

1. **Given** the health checker source code, **When** a developer reads the Check function, **Then** they should find clear documentation explaining the goroutine leak limitation and why it's acceptable.

---

### User Story 4 - Performance-Sensitive Deployment (Priority: P3)

A deployment with frequent logging needs efficient logging that doesn't create unnecessary allocations on every log call.

**Why this priority**: While logging overhead is typically negligible, eliminating unnecessary allocations is good practice and improves performance at scale.

**Independent Test**: Can be verified through code review confirming handlers are cached rather than recreated per log call.

**Acceptance Scenarios**:

1. **Given** the multiStreamHandler implementation, **When** multiple log calls are made, **Then** the same pre-created slog handlers should be reused rather than creating new ones.

---

### Edge Cases

- What happens when config file is exactly 1MB? Should be accepted (limit is exclusive >1MB).
- How does system handle world-writable check on Windows? Should be skipped (Windows permissions work differently).
- What happens if the permission check fails (e.g., stat errors)? Should continue loading (warning-only approach).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST reject JSON config files larger than 1 megabyte with a clear error message indicating the size limit.
- **FR-002**: System MUST warn (via log) if config file has world-writable permissions on Unix systems, but continue loading.
- **FR-003**: System MUST skip world-writable permission check on Windows platforms.
- **FR-004**: System MUST provide config validation error messages that explain WHY a constraint exists, not just WHAT is wrong.
- **FR-005**: System MUST document the goroutine leak limitation in the health checker code with explanation of why it's acceptable.
- **FR-006**: System MUST cache slog handlers in the multiStreamHandler to avoid per-log-call allocations.

### Issues to Close (No Implementation)

The following issues will be closed as "won't do" with explanatory comments:

- **#16** - Log warning when env var parsing fails (silent fallback is acceptable)
- **#14** - Add migration guide env vars to JSON (README is sufficient)
- **#13** - Make mount.Name access consistent (Name is immutable, code is clear)
- **#11** - Implement parallel mount checking (over-engineering)
- **#10** - Add Kubernetes manifest examples (out of scope)
- **#9** - Add missing godoc comments (internal packages don't need public docs)

### Key Entities *(include if feature involves data)*

- **Config File**: JSON file containing mount monitor configuration. Now has 1MB size limit and permission warnings.
- **multiStreamHandler**: Custom slog.Handler routing logs by level. Now caches handlers for efficiency.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All 5 implementation issues (#7, #8, #12, #15, #17) are resolved with passing tests.
- **SC-002**: All 6 "won't do" issues (#9, #10, #11, #13, #14, #16) are closed with explanatory comments.
- **SC-003**: Existing test suite passes with no regressions.
- **SC-004**: Config files over 1MB are rejected at startup.
- **SC-005**: World-writable config files generate a warning log on Unix systems.
