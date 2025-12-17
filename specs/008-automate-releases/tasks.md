# Tasks: Automate Releases

**Input**: Design documents from `/specs/008-automate-releases/`
**Prerequisites**: plan.md, spec.md, contracts/release-workflow.yaml, research.md, quickstart.md

**Tests**: No automated tests requested for this CI/CD workflow feature. Validation is done via manual test release.

**Organization**: Tasks create a single workflow file (`.github/workflows/release.yml`) with components organized by user story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- All tasks modify `.github/workflows/release.yml` unless otherwise noted

---

## Phase 1: Setup (Workflow Scaffolding)

**Purpose**: Create the release workflow file with triggers and permissions

- [ ] T001 Create workflow file with name and trigger configuration in `.github/workflows/release.yml`
- [ ] T002 Add workflow_dispatch input for manual releases in `.github/workflows/release.yml`
- [ ] T003 Configure permissions (contents: write, packages: write) in `.github/workflows/release.yml`
- [ ] T004 Define environment variables (GO_VERSION, REGISTRY, IMAGE_NAME) in `.github/workflows/release.yml`

**Checkpoint**: Workflow file exists with proper triggers and permissions

---

## Phase 2: Foundational (Test Gate Job)

**Purpose**: Implement the test job that gates all release artifacts (FR-012)

**⚠️ CRITICAL**: This job must pass before any artifacts are created

- [ ] T005 Implement `test` job with Go setup and test execution in `.github/workflows/release.yml`
- [ ] T006 Add race detection flag to test command in `.github/workflows/release.yml`

**Checkpoint**: Test gate job complete - pushing a tag runs tests first

---

## Phase 3: User Story 1 - Create Tagged Release (Priority: P1) 🎯 MVP

**Goal**: Maintainers can create releases by pushing semantic version tags

**Independent Test**: Push tag `v0.0.1-test` to fork, verify workflow triggers and completes

### Implementation for User Story 1

- [ ] T007 [US1] Add version detection step (tag vs workflow_dispatch) in `.github/workflows/release.yml`
- [ ] T008 [US1] Add pre-release detection regex (contains `-` after semver) in `.github/workflows/release.yml`
- [ ] T009 [US1] Implement `build` job with matrix strategy (linux/amd64, linux/arm64) in `.github/workflows/release.yml`
- [ ] T010 [US1] Add version injection via LDFLAGS in build job in `.github/workflows/release.yml`
- [ ] T011 [US1] Configure CGO_ENABLED=0 for static linking in `.github/workflows/release.yml`
- [ ] T012 [US1] Add artifact upload step for built binaries in `.github/workflows/release.yml`
- [ ] T013 [US1] Add `needs: test` dependency to build job in `.github/workflows/release.yml`

**Checkpoint**: Tag push triggers workflow → tests run → binaries build for both architectures

---

## Phase 4: User Story 2 - Download Pre-built Binaries (Priority: P2)

**Goal**: Users can download versioned binaries from GitHub Releases with checksums

**Independent Test**: After release, download binary and verify `./mount-monitor-linux-amd64 --help` shows correct version

### Implementation for User Story 2

- [ ] T014 [US2] Implement `verify` job to test binary with --help in `.github/workflows/release.yml`
- [ ] T015 [US2] Add `needs: build` dependency to verify job in `.github/workflows/release.yml`
- [ ] T016 [US2] Implement `release` job scaffold in `.github/workflows/release.yml`
- [ ] T017 [US2] Add artifact download step in release job in `.github/workflows/release.yml`
- [ ] T018 [US2] Add SHA256 checksum generation step in release job in `.github/workflows/release.yml`
- [ ] T019 [US2] Add softprops/action-gh-release step with binary uploads in `.github/workflows/release.yml`
- [ ] T020 [US2] Configure pre-release flag based on version detection in `.github/workflows/release.yml`

**Checkpoint**: GitHub Release created with binaries and checksums.txt

---

## Phase 5: User Story 3 - Pull Docker Image (Priority: P2)

**Goal**: K8s operators can pull versioned Docker images from ghcr.io

**Independent Test**: After release, run `docker pull ghcr.io/cscheib/debrid-mount-monitor:<version>` and verify image works

### Implementation for User Story 3

- [ ] T021 [US3] Implement `docker` job scaffold in `.github/workflows/release.yml`
- [ ] T022 [US3] Add docker/setup-qemu-action for multi-arch in `.github/workflows/release.yml`
- [ ] T023 [US3] Add docker/setup-buildx-action in `.github/workflows/release.yml`
- [ ] T024 [US3] Add docker/login-action for ghcr.io authentication in `.github/workflows/release.yml`
- [ ] T025 [US3] Add nick-fields/retry@v3 wrapper for Docker build/push in `.github/workflows/release.yml`
- [ ] T026 [US3] Configure multi-platform build (linux/amd64,linux/arm64) in `.github/workflows/release.yml`
- [ ] T027 [US3] Implement conditional `latest` tag (stable releases only) in `.github/workflows/release.yml`
- [ ] T028 [US3] Add `needs: verify` dependency to docker job in `.github/workflows/release.yml`

**Checkpoint**: Docker images pushed to ghcr.io with version tag (and `latest` for stable)

---

## Phase 6: User Story 4 - View Release Notes (Priority: P3)

**Goal**: Users see auto-generated changelog on GitHub Release page

**Independent Test**: View release on GitHub, verify commit history is summarized

### Implementation for User Story 4

- [ ] T029 [US4] Enable `generate_release_notes: true` in softprops/action-gh-release in `.github/workflows/release.yml`
- [ ] T030 [US4] Add `needs: docker` dependency to release job (final ordering) in `.github/workflows/release.yml`

**Checkpoint**: Release notes auto-generated from commits since last tag

---

## Phase 7: Polish & Validation

**Purpose**: Final validation and documentation

- [ ] T031 [P] Review workflow against contracts/release-workflow.yaml for completeness
- [ ] T032 [P] Validate all FR requirements (FR-001 through FR-016) are implemented
- [ ] T033 [P] Update quickstart.md with actual repository name if needed in `specs/008-automate-releases/quickstart.md`
- [ ] T034 Test workflow by pushing test tag to fork (manual validation)
- [ ] T035 Remove test release/tag after validation

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1: Setup           → No dependencies
Phase 2: Foundational    → Depends on Setup (T001-T004)
Phase 3: US1 (P1)        → Depends on Foundational (T005-T006)
Phase 4: US2 (P2)        → Depends on US1 build job (T009-T013)
Phase 5: US3 (P2)        → Depends on US2 verify job (T014-T015)
Phase 6: US4 (P3)        → Depends on US3 docker job (T021-T028)
Phase 7: Polish          → Depends on all user stories
```

### User Story Dependencies

All user stories contribute to a single workflow file, so they build sequentially:

- **US1 (P1)**: Core trigger/build - foundation for all other stories
- **US2 (P2)**: Adds release job with binaries - depends on US1 build artifacts
- **US3 (P2)**: Adds docker job - depends on US2 verify job
- **US4 (P3)**: Enhances release job - depends on US3 ordering

### Parallel Opportunities

Limited parallelism due to single-file workflow:

- T031, T032, T033 can run in parallel (different files/review tasks)
- Within each phase, tasks are sequential (same file modifications)

---

## Parallel Example: Phase 7 Polish

```bash
# These can run in parallel (different focuses):
Task: "Review workflow against contracts/release-workflow.yaml"
Task: "Validate all FR requirements are implemented"
Task: "Update quickstart.md with actual repository name"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T004)
2. Complete Phase 2: Foundational (T005-T006)
3. Complete Phase 3: User Story 1 (T007-T013)
4. **STOP and VALIDATE**: Push test tag, verify workflow triggers and builds binaries
5. This provides working release automation without GitHub Release or Docker

### Incremental Delivery

1. Setup + Foundational + US1 → Workflow builds binaries on tag push (MVP!)
2. Add US2 → GitHub Release with downloadable binaries + checksums
3. Add US3 → Docker images on ghcr.io
4. Add US4 → Auto-generated release notes
5. Each addition is testable by pushing a new test tag

### Single Developer Strategy

All tasks are in a single workflow file, so sequential implementation is natural:

1. Build the workflow file section by section
2. Test after each phase by pushing a test tag
3. Clean up test releases before final validation

---

## Task Summary

| Phase | Tasks | User Story | Description |
|-------|-------|------------|-------------|
| 1 | T001-T004 | Setup | Workflow triggers and permissions |
| 2 | T005-T006 | Foundational | Test gate job |
| 3 | T007-T013 | US1 (P1) | Tag triggers and binary builds |
| 4 | T014-T020 | US2 (P2) | GitHub Release with binaries |
| 5 | T021-T028 | US3 (P2) | Docker image publishing |
| 6 | T029-T030 | US4 (P3) | Release notes generation |
| 7 | T031-T035 | Polish | Validation and cleanup |

**Total Tasks**: 35
**MVP Tasks** (US1 only): 13 (T001-T013)

---

## Notes

- All implementation tasks modify `.github/workflows/release.yml`
- Test by pushing tags like `v0.0.1-test` to a fork
- Delete test releases/tags after validation
- Contract file (`contracts/release-workflow.yaml`) serves as reference implementation
- FR-016 (retry) implemented via nick-fields/retry@v3 in T025
