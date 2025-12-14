<!--
Sync Impact Report
==================
Version change: 1.0.0 → 1.1.0 (MINOR - new section and principle added)

Modified principles: None

Added principles:
- VI. Fail-Safe Orchestration

Added sections:
- Project Purpose & Domain (new top-level section)

Removed sections: None

Templates requiring updates:
- .specify/templates/plan-template.md ✅ (compatible - no changes required)
- .specify/templates/spec-template.md ✅ (compatible - no changes required)
- .specify/templates/tasks-template.md ✅ (compatible - no changes required)

Follow-up TODOs: None
==================
-->

# Debrid Mount Monitor Constitution

## Project Purpose & Domain

This service ensures the health of debrid WebDAV mounts, which are critical infrastructure for the proper functionality of a media server ecosystem (e.g., Plex, Jellyfin, Emby with cloud-based media storage).

**Core Responsibilities**:
- Monitor the health and availability of debrid WebDAV mount points
- Detect mount failures, stale mounts, or connectivity issues
- Restart dependent services when mounts become unhealthy
- Gate service startup to prevent dependent services from starting until mounts are healthy

**Domain Context**: Debrid services provide cloud storage accessible via WebDAV. Media servers depend on these mounts being available and responsive. Mount failures can cause media servers to index empty directories, corrupt metadata, or fail to serve content. This monitor acts as a health gate and recovery mechanism.

## Core Principles

### I. Minimal Dependencies

All code MUST minimize external dependencies. Every dependency added MUST be justified with a clear rationale explaining why the functionality cannot be implemented with standard library primitives.

**Rationale**: Fewer dependencies reduce attack surface, simplify auditing, decrease binary size, and eliminate transitive dependency risks. A dependency-light codebase is easier to maintain and has fewer breaking changes from upstream.

**Compliance**:
- Standard library solutions MUST be preferred over third-party packages
- Any external dependency MUST be documented with justification
- Dependencies MUST NOT pull in large transitive dependency trees

### II. Single Static Binary

The build output MUST be a single, statically-linked binary with no runtime dependencies. The binary MUST execute in a minimal container environment (e.g., `scratch`, `distroless`) without requiring additional libraries, interpreters, or runtime components.

**Rationale**: A single static binary simplifies deployment, reduces container image size, eliminates "works on my machine" issues, and ensures consistent behavior across environments.

**Compliance**:
- Build configuration MUST produce a statically-linked executable
- The binary MUST NOT require shared libraries at runtime
- The binary MUST NOT require configuration files to start (configuration via environment variables or flags)
- Container images MUST be buildable from `scratch` or equivalent minimal base

### III. Cross-Platform Compilation

The project MUST compile successfully for both ARM (aarch64) and x86-64 (amd64) architectures. Build tooling MUST support cross-compilation from any development platform.

**Rationale**: Modern infrastructure spans multiple architectures (cloud ARM instances, Apple Silicon development machines, traditional x86 servers). Cross-platform support ensures deployment flexibility and developer productivity.

**Compliance**:
- CI/CD pipelines MUST build and test for both architectures
- Architecture-specific code MUST be clearly isolated and documented
- Release artifacts MUST include binaries for both ARM and x86-64

### IV. Signal Handling

The service MUST respond correctly to standard UNIX process signals:
- **SIGTERM**: Initiate graceful shutdown with cleanup
- **SIGINT**: Initiate graceful shutdown (interactive termination)
- **SIGHUP**: Reload configuration if applicable (or treat as SIGTERM)

**Rationale**: Container orchestrators (Kubernetes, Docker) rely on signal handling for lifecycle management. Proper signal handling ensures graceful shutdowns, prevents data corruption, and enables zero-downtime deployments.

**Compliance**:
- Signal handlers MUST be registered at startup
- Graceful shutdown MUST complete within a reasonable timeout (default: 30 seconds)
- In-flight operations MUST be allowed to complete or be cleanly cancelled
- Exit codes MUST follow convention (0 for success, non-zero for errors)

### V. Container Sidecar Design

The service is designed to operate as a sidecar container alongside a primary application container. Design decisions MUST account for the sidecar deployment pattern.

**Rationale**: Sidecar containers have specific constraints: shared network namespace, ephemeral storage, coordinated lifecycle with the main container, and minimal resource overhead.

**Compliance**:
- Resource usage (CPU, memory) MUST be minimal and bounded
- Startup MUST be fast to avoid delaying pod readiness
- Health check endpoints MUST be provided if the service exposes network interfaces
- Logging MUST go to stdout/stderr (not files) for container log aggregation
- The service MUST NOT assume persistent local storage

### VI. Fail-Safe Orchestration

When mount health degrades, the service MUST take protective action to prevent dependent services from operating in an unhealthy state. The default behavior MUST be fail-safe: prefer service unavailability over data corruption or incorrect operation.

**Rationale**: Media servers operating against failed mounts can corrupt metadata databases, index empty directories as "missing" content, or serve incomplete data. It is better to stop services and wait for recovery than to allow degraded operation.

**Compliance**:
- Health checks MUST detect mount failures within a configurable timeout
- Dependent service restarts MUST be triggered when unhealthy state is detected
- Service startup gating MUST prevent dependent services from starting until mounts are verified healthy
- Recovery actions MUST be idempotent (safe to retry)
- The service MUST log all health state transitions and orchestration actions
- False positive protection: brief transient failures SHOULD NOT trigger restarts (configurable debounce/threshold)

## Build & Distribution Requirements

**Language Selection**: The implementation language MUST support:
- Static linking without runtime dependencies
- Cross-compilation for ARM and x86-64
- Efficient signal handling
- Low memory footprint

Recommended languages that satisfy these requirements: Go, Rust, C, Zig.

**Container Image Requirements**:
- Production images MUST use minimal base images (`scratch`, `distroless`, or Alpine)
- Multi-architecture manifests MUST be published for container registries
- Image size SHOULD be minimized (target: < 20MB uncompressed)

## Runtime Behavior

**Configuration**: All configuration MUST be injectable via:
- Environment variables (primary method for containers)
- Command-line flags (for development and testing)

Configuration files MAY be supported but MUST NOT be required.

**Observability**:
- Structured logging (JSON) MUST be supported
- Logs MUST be written to stdout (info, debug) and stderr (errors, warnings)
- Metrics endpoints MAY be exposed if monitoring integration is required

**Error Handling**:
- Errors MUST be logged with sufficient context for debugging
- Fatal errors MUST result in non-zero exit codes
- The service MUST NOT silently fail or hang

## Governance

This constitution defines the non-negotiable architectural constraints for the debrid-mount-monitor project. All implementation decisions, pull requests, and code reviews MUST verify compliance with these principles.

**Amendment Process**:
1. Proposed changes MUST be documented with rationale
2. Changes MUST be reviewed for impact on existing code
3. Version number MUST be incremented according to semantic versioning:
   - MAJOR: Removal or incompatible redefinition of principles
   - MINOR: Addition of new principles or sections
   - PATCH: Clarifications and non-semantic refinements

**Compliance Review**:
- All PRs MUST pass constitution compliance checks
- Violations MUST be justified in the Complexity Tracking section of the implementation plan
- Unjustified violations are grounds for PR rejection

**Version**: 1.1.0 | **Ratified**: 2025-12-14 | **Last Amended**: 2025-12-14
