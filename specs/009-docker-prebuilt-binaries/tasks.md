# Tasks: Docker Images Use Pre-built Binaries

**Input**: Design documents from `/specs/009-docker-prebuilt-binaries/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/ ✓

**Tests**: Manual workflow test only (no unit tests - CI/CD infrastructure change)

**Organization**: This feature is atomic - all three user stories (Faster Pipeline, Version Embedding, Multi-arch Support) are delivered by the same implementation. Tasks are organized by artifact.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to
- Include exact file paths in descriptions

## User Stories Summary

| Story | Priority | Delivered By |
|-------|----------|--------------|
| US1: Faster Release Pipeline | P1 | T001-T003 (Dockerfile.release + workflow changes) |
| US2: Consistent Version Embedding | P1 | T001-T003 (pre-built binaries have version embedded) |
| US3: Multi-arch Image Support | P2 | T001-T003 (buildx with TARGETARCH) |

All stories are delivered atomically by the same implementation.

---

## Phase 1: Setup

**Purpose**: No setup required - modifying existing project

This feature modifies existing infrastructure. No project initialization needed.

**Checkpoint**: Ready to proceed directly to implementation

---

## Phase 2: Implementation (All User Stories)

**Purpose**: Create Dockerfile.release and update workflow to use pre-built binaries

**Goal**: Docker images use pre-compiled binaries from build job instead of recompiling

**Independent Test**: Push test tag, verify Docker job logs show no `go build` commands

### Implementation

- [x] T001 [P] [US1/US2/US3] Create Dockerfile.release at repository root from specs/009-docker-prebuilt-binaries/contracts/Dockerfile.release

- [x] T002 [US1/US2/US3] Update Docker job in .github/workflows/release.yml:
  - Add download steps for amd64 and arm64 binaries to ./docker-context/
  - Add sparse checkout for Dockerfile.release only
  - Add step to copy Dockerfile.release to ./docker-context/
  - Update buildx command to use --file ./docker-context/Dockerfile.release and ./docker-context/ as build context

- [x] T003 [US1/US2/US3] Verify local development workflow still works:
  - Run `make docker` to confirm existing Dockerfile is used
  - Run `docker run mount-monitor:dev --help` to confirm image works

**Checkpoint**: Implementation complete - all user stories delivered

---

## Phase 3: Verification

**Purpose**: Validate the implementation meets all acceptance criteria

### Manual Verification Steps

- [ ] T004 Create and push test tag to trigger release workflow:
  - `git add Dockerfile.release .github/workflows/release.yml`
  - `git commit -m "feat: use pre-built binaries for Docker images"`
  - `git tag v0.0.1-test`
  - `git push origin 009-docker-prebuilt-binaries v0.0.1-test`

- [ ] T005 [US1] Verify no Go compilation in Docker job:
  - Check GitHub Actions workflow logs
  - Confirm no `go build` or `go mod download` in Docker job logs
  - Confirm Docker build time is reduced

- [ ] T006 [US2] Verify version embedding in Docker image:
  - `docker pull ghcr.io/cscheib/debrid-mount-monitor:v0.0.1-test`
  - `docker run ghcr.io/cscheib/debrid-mount-monitor:v0.0.1-test --help`
  - Confirm version shows `0.0.1-test`

- [ ] T007 [US3] Verify multi-arch manifest:
  - `docker manifest inspect ghcr.io/cscheib/debrid-mount-monitor:v0.0.1-test`
  - Confirm both linux/amd64 and linux/arm64 platforms listed

- [ ] T008 Cleanup test tag:
  - `git push origin :refs/tags/v0.0.1-test`
  - Delete pre-release from GitHub Releases if created

**Checkpoint**: All acceptance criteria verified

---

## Phase 4: Polish & Documentation

**Purpose**: Final cleanup and documentation

- [x] T009 [P] Update spec status from Draft to Complete in specs/009-docker-prebuilt-binaries/spec.md

- [x] T010 [P] Add comment to existing Dockerfile explaining relationship to Dockerfile.release

**Checkpoint**: Feature complete and documented

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1: Setup (skipped - no setup needed)
    │
    ▼
Phase 2: Implementation
    │ T001 (Dockerfile.release) ──┐
    │ T002 (workflow update)  ────┼──► T003 (local dev check)
    │                             │
    ▼                             │
Phase 3: Verification ◄───────────┘
    │ T004 (push test tag)
    │   │
    │   ▼
    │ T005 (check logs) ──┐
    │ T006 (check version)┼──► T008 (cleanup)
    │ T007 (check manifest)┘
    │
    ▼
Phase 4: Polish
    │ T009 (update spec)
    │ T010 (add comment)
```

### Parallel Opportunities

- **T001 and T002**: Can be drafted in parallel (different files)
- **T005, T006, T007**: Can run in parallel after T004 completes
- **T009 and T010**: Can run in parallel (different files)

---

## Parallel Example

```bash
# Implementation (can draft in parallel, commit sequentially):
Task: "Create Dockerfile.release at repository root"
Task: "Update Docker job in .github/workflows/release.yml"

# Verification (run in parallel after push):
Task: "Verify no Go compilation in Docker job"
Task: "Verify version embedding in Docker image"
Task: "Verify multi-arch manifest"

# Polish (run in parallel):
Task: "Update spec status from Draft to Complete"
Task: "Add comment to existing Dockerfile"
```

---

## Implementation Strategy

### Atomic Delivery

This feature is atomic - all three user stories succeed or fail together:
1. Complete T001 (Dockerfile.release)
2. Complete T002 (workflow update)
3. Verify T003 (local dev still works)
4. Push and verify with test tag (T004-T007)
5. Cleanup and polish (T008-T010)

### Rollback Strategy

If verification fails:
1. Revert workflow changes in release.yml
2. Delete Dockerfile.release
3. Investigate failure cause
4. Re-attempt after fix

### Time Estimate

| Phase | Estimated Time |
|-------|----------------|
| Implementation (T001-T003) | 15 minutes |
| Verification (T004-T008) | 20 minutes (includes CI wait) |
| Polish (T009-T010) | 5 minutes |
| **Total** | ~40 minutes |

---

## Notes

- All user stories delivered by same implementation (atomic)
- No unit tests needed - CI/CD infrastructure change
- Manual verification via test tag push
- Local development workflow (`make docker`) unchanged
- Existing Dockerfile preserved for backward compatibility
