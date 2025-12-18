# Feature Specification: Docker Images Use Pre-built Binaries

**Feature Branch**: `009-docker-prebuilt-binaries`
**Created**: 2025-12-17
**Status**: Complete
**Input**: User description: "the docker images shouldn't recompile the binary - they should utilize the already compiled binary artifacts from earlier in the pipeline"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Faster Release Pipeline (Priority: P1)

As a maintainer, I want Docker images to use pre-compiled binaries from earlier in the release pipeline so that releases complete faster and CI resources are used efficiently.

**Why this priority**: This is the core value proposition - eliminating redundant compilation saves time and compute resources on every release.

**Independent Test**: Can be tested by running the release workflow and observing that the Docker build step completes without invoking Go compilation (no `go build` in logs).

**Acceptance Scenarios**:

1. **Given** the build job has completed and produced binary artifacts, **When** the Docker job runs, **Then** it downloads and uses those pre-built binaries instead of compiling from source.
2. **Given** the release workflow runs, **When** all jobs complete, **Then** the total pipeline time is reduced compared to the previous approach that compiled twice.
3. **Given** the Docker job starts, **When** building images, **Then** no Go toolchain is invoked (no `go build` commands in logs).

---

### User Story 2 - Consistent Version Embedding (Priority: P1)

As a user pulling Docker images, I want the container to report the correct version so that I can verify I'm running the expected release.

**Why this priority**: Currently Docker images don't have the version embedded (the Dockerfile builds without `-X main.Version`). This fixes a gap where binaries from GitHub Releases have versions but Docker images don't.

**Independent Test**: Can be tested by pulling a Docker image and running `--help` or `--version` to verify the embedded version matches the release tag.

**Acceptance Scenarios**:

1. **Given** a release tag `v1.2.3` is pushed, **When** a user runs the Docker image with `--help`, **Then** the version displayed is `1.2.3`.
2. **Given** binaries are built with version embedding in the build job, **When** those binaries are copied into Docker images, **Then** the version is preserved.

---

### User Story 3 - Multi-architecture Image Support (Priority: P2)

As a Kubernetes operator deploying on mixed ARM64/AMD64 clusters, I want Docker images for both architectures to use the correct pre-built binary so that deployment works seamlessly on any node.

**Why this priority**: Must maintain existing multi-arch support while changing the build approach.

**Independent Test**: Can be tested by pulling the image on both AMD64 and ARM64 systems and verifying the correct binary runs.

**Acceptance Scenarios**:

1. **Given** binaries exist for linux/amd64 and linux/arm64, **When** the Docker build creates a multi-arch manifest, **Then** each architecture uses its corresponding pre-built binary.
2. **Given** a user on ARM64 pulls the image, **When** they run the container, **Then** the ARM64 binary executes (not amd64 under emulation).

---

### Edge Cases

- What happens if the build job fails and binaries aren't available? → Docker job depends on build job; it won't run if binaries don't exist.
- What happens if only one architecture's binary is available? → Docker job should fail; both binaries are required for multi-arch manifest.
- What happens if artifact download fails? → Job fails; existing retry mechanism handles transient failures.
- What happens with local development builds? → Developers can still use `make docker` which uses the original multi-stage Dockerfile.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST download pre-built binary artifacts from the build job before creating Docker images.
- **FR-002**: System MUST NOT compile Go source code during Docker image creation in the release workflow.
- **FR-003**: System MUST use a Dockerfile that copies in a pre-built binary rather than building from source.
- **FR-004**: System MUST create Docker images for linux/amd64 and linux/arm64, each containing the architecture-appropriate binary.
- **FR-005**: System MUST maintain the existing multi-architecture manifest so users can pull a single tag that resolves to the correct architecture.
- **FR-006**: System MUST preserve version embedding in Docker images by using binaries that have version compiled in.
- **FR-007**: System MUST maintain backward compatibility with local development Docker builds (existing Dockerfile unchanged for `make docker`).
- **FR-008**: System MUST keep the final Docker image minimal (using `scratch` base).
- **FR-009**: System MUST maintain existing Docker image tags (version tag + `latest` for stable releases).

### Key Entities

- **Binary Artifact**: Pre-compiled executable uploaded by the build job, one per target architecture (linux/amd64, linux/arm64).
- **Release Dockerfile**: A Dockerfile variant optimized for copying pre-built binaries (no builder stage with Go toolchain).
- **Multi-arch Manifest**: Docker manifest list that maps a single tag to multiple architecture-specific images.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Docker build step completes without invoking Go compiler (no `go build` in job logs).
- **SC-002**: Docker images display correct version matching the release tag when run with `--help`.
- **SC-003**: Total release pipeline time decreases by eliminating redundant compilation step.
- **SC-004**: Docker images for both linux/amd64 and linux/arm64 continue to be published to ghcr.io.
- **SC-005**: Docker image size remains unchanged (same `scratch` base, same binary).
- **SC-006**: Local development workflows using `make docker` continue to work for developers.

## Assumptions

- The existing build job produces functional, statically-linked binaries for linux/amd64 and linux/arm64.
- GitHub Actions artifact storage reliably passes binaries between jobs (already proven in current workflow).
- The `docker buildx` tool supports building images from pre-built binaries without multi-stage builds.
- Maintaining a separate release Dockerfile is acceptable (keeps local dev workflow simple).
- The existing retry mechanism (3 attempts, 30-second wait) handles transient Docker registry failures.

## Out of Scope

- Changes to which architectures are supported (remains linux/amd64 and linux/arm64).
- Changes to the Docker registry (remains ghcr.io).
- Changes to tagging conventions (remains version tag + latest for stable).
- Changes to the GitHub Release creation process.
- Optimization of the binary build job itself.
- Changes to Dockerfile.debug (can be addressed separately if needed).
