# Feature Specification: Automate Releases

**Feature Branch**: `008-automate-releases`
**Created**: 2025-12-17
**Status**: Draft
**Input**: User description: "automate releases - publish binary and docker image to registry when a release is required"

## Clarifications

### Session 2025-12-17

- Q: How should maintainers be notified of release failures? → A: GitHub default (email to committer/watchers on failure)
- Q: Should the workflow automatically retry transient failures? → A: Yes, auto-retry transient failures (2 retries per step)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create Tagged Release (Priority: P1)

As a maintainer, I want to create a new release by pushing a semantic version tag (e.g., `v1.2.0`) so that binaries and Docker images are automatically built and published without manual intervention.

**Why this priority**: This is the core value proposition - eliminating manual release steps and ensuring consistent, reproducible releases.

**Independent Test**: Can be tested by pushing a tag like `v0.1.0-test` to a fork and verifying that release artifacts appear on GitHub Releases and Docker images are pushed to ghcr.io.

**Acceptance Scenarios**:

1. **Given** the main branch is stable and tested, **When** a maintainer pushes a tag matching `v*` (e.g., `v1.0.0` or `v1.0.0-beta.1`), **Then** the release workflow triggers automatically.
2. **Given** the release workflow is triggered, **When** all builds complete successfully, **Then** pre-compiled binaries for Linux amd64 and arm64 are uploaded to GitHub Releases.
3. **Given** the release workflow is triggered, **When** all builds complete successfully, **Then** multi-architecture Docker images are pushed to ghcr.io.
4. **Given** a tag is pushed, **When** CI tests fail, **Then** the release workflow does not proceed and maintainers are notified of the failure.
5. **Given** a pre-release tag is pushed (e.g., `v1.0.0-beta.1`), **When** the release is created, **Then** it is marked as a pre-release on GitHub and the `latest` Docker tag is not updated.

---

### User Story 2 - Download Pre-built Binaries (Priority: P2)

As a user, I want to download pre-compiled binaries from GitHub Releases so that I can install the monitor without needing Go installed or building from source.

**Why this priority**: Makes the project accessible to users who don't have Go development environments, expanding the user base.

**Independent Test**: Can be tested by navigating to GitHub Releases page and downloading a binary for the user's platform, then verifying it runs with `--help`.

**Acceptance Scenarios**:

1. **Given** a release exists, **When** a user visits the GitHub Releases page, **Then** they see downloadable binaries for linux-amd64 and linux-arm64.
2. **Given** a user downloads a binary, **When** they run `./mount-monitor --help`, **Then** the application displays help text with the correct version number.
3. **Given** a release exists, **When** a user downloads a binary, **Then** the binary is statically linked and runs without additional dependencies.
4. **Given** a release exists, **When** a user downloads the checksums file, **Then** they can verify binary integrity using SHA256.

---

### User Story 3 - Pull Docker Image from Registry (Priority: P2)

As a Kubernetes operator, I want to pull versioned Docker images from GitHub Container Registry so that I can deploy specific versions of the monitor to my clusters.

**Why this priority**: Enables production deployments with version pinning, essential for Kubernetes users who are the primary audience.

**Independent Test**: Can be tested by running `docker pull ghcr.io/<owner>/mount-monitor:v1.0.0` and verifying the image runs correctly.

**Acceptance Scenarios**:

1. **Given** a release is published, **When** a user runs `docker pull ghcr.io/<owner>/mount-monitor:<version>`, **Then** the image is downloaded successfully.
2. **Given** a user pulls the image, **When** they run `docker run ghcr.io/<owner>/mount-monitor:<version> --help`, **Then** the application shows the correct version.
3. **Given** multiple stable releases exist, **When** a user pulls `ghcr.io/<owner>/mount-monitor:latest`, **Then** they receive the most recent stable release (not pre-release).
4. **Given** the image is multi-architecture, **When** a user on ARM64 pulls the image, **Then** they automatically receive the ARM64 variant.

---

### User Story 4 - View Release Notes (Priority: P3)

As a user, I want to see what changed in each release so that I can decide whether to upgrade and understand any breaking changes.

**Why this priority**: Good for user communication but not blocking for the core release automation.

**Independent Test**: Can be tested by viewing a GitHub Release and verifying changelog content is present.

**Acceptance Scenarios**:

1. **Given** a new release is created, **When** a user views the GitHub Release page, **Then** they see a changelog summarizing commits since the last release.

---

### Edge Cases

- What happens when a tag is pushed but CI tests fail? → Release workflow should not proceed; no artifacts published.
- What happens if Docker registry push fails mid-release? → GitHub Release should still be created with binaries; failure should be clearly reported.
- What happens if someone pushes a non-semver tag (e.g., `test-tag`)? → Release workflow should ignore non-matching tags.
- What happens if someone re-pushes an existing tag? → Workflow should fail gracefully (cannot overwrite existing release).
- What happens if the release workflow is run manually? → Should support manual trigger with version input for emergency releases.
- What happens with pre-release tags? → Marked as pre-release on GitHub, `latest` Docker tag not updated.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST trigger release workflow only when a Git tag matching semantic versioning pattern (v*) is pushed, including pre-release versions.
- **FR-002**: System MUST build statically-linked binaries for Linux amd64 and Linux arm64 architectures.
- **FR-003**: System MUST embed the version from the Git tag into the binary at build time.
- **FR-004**: System MUST create a GitHub Release with the tag name as the release title.
- **FR-005**: System MUST upload compiled binaries as release assets with platform-specific naming (e.g., `mount-monitor-linux-amd64`, `mount-monitor-linux-arm64`).
- **FR-006**: System MUST generate and upload a SHA256 checksums file for all binary artifacts.
- **FR-007**: System MUST build multi-architecture Docker images supporting linux/amd64 and linux/arm64.
- **FR-008**: System MUST push Docker images to GitHub Container Registry (ghcr.io) with the version tag.
- **FR-009**: System MUST update the `latest` Docker tag only for stable releases (not pre-releases).
- **FR-010**: System MUST mark pre-release versions (containing `-`) as pre-releases on GitHub.
- **FR-011**: System MUST generate release notes based on commit history since the last release.
- **FR-012**: System MUST NOT proceed with releases if CI tests fail.
- **FR-013**: System MUST support manual workflow trigger for emergency releases with version input.
- **FR-014**: System MUST use the existing Dockerfile for building release images.
- **FR-015**: System MUST verify binary artifacts are functional by running `--help` before publishing.
- **FR-016**: System MUST automatically retry failed steps up to 2 times to handle transient failures (e.g., network timeouts, registry unavailability).

### Key Entities

- **Release**: A versioned distribution of the software, identified by a semantic version tag, containing binaries and Docker images.
- **Binary Artifact**: A compiled, statically-linked executable for a specific platform/architecture combination.
- **Docker Image**: A container image built from the project's Dockerfile, tagged with version and architecture metadata.
- **Release Notes**: Auto-generated changelog describing changes since the previous release.
- **Checksums File**: A text file containing SHA256 hashes for all binary artifacts.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Maintainers can create a complete release (binaries + Docker images + release notes) by pushing a single Git tag, with no manual steps required.
- **SC-002**: Users can download and run binaries on supported platforms within 5 minutes of release publication.
- **SC-003**: Kubernetes operators can deploy new versions by changing the image tag in their manifests, with images available within 15 minutes of tag push.
- **SC-004**: 100% of releases include both linux/amd64 and linux/arm64 binaries and Docker images.
- **SC-005**: All published binaries report the correct version when run with `--help`.
- **SC-006**: Failed CI runs prevent release publication 100% of the time.
- **SC-007**: Release notes accurately reflect commits since the previous tagged release.
- **SC-008**: Pre-release versions are correctly identified and do not update the `latest` Docker tag.
- **SC-009**: All releases include SHA256 checksums for binary verification.

## Assumptions

- GitHub Actions is the CI/CD platform (based on existing `.github/workflows/` configuration).
- GitHub Container Registry (ghcr.io) will be used for Docker images.
- Semantic versioning (v*.*.*) with optional pre-release suffixes is the tagging convention.
- Only Linux platforms are required (darwin/windows not needed based on Kubernetes-focused use case).
- The existing Makefile build targets will be leveraged for consistency.
- GitHub's automatic release notes generation will be sufficient for release notes.
- The repository owner/organization name is `cscheib` (from go.mod module path).
- Release failure notifications rely on GitHub's default behavior (email to committer and repository watchers); no additional notification integrations required.

## Out of Scope

- macOS or Windows binary builds
- Code signing for binaries
- Publishing to package managers (apt, yum, brew)
- Automated version bumping or changelog management
- Notifications to external services (Slack, Discord, etc.)
- Docker Hub publishing (using ghcr.io only)
