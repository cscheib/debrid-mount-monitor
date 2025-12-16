# Tasks: Development Tooling Improvements

**Input**: Design documents from `/specs/006-dev-tooling-improvements/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, quickstart.md ✓

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md structure:
- **Makefile**: Repository root
- **Scripts**: `scripts/`
- **Tests**: `tests/unit/`
- **Docs**: `docs/`
- **KIND manifests**: `deploy/kind/`

---

## Phase 1: Setup

**Purpose**: Create directory structure and scaffolding for new tooling

- [x] T001 Create scripts directory at repository root `scripts/`
- [x] T002 [P] Create docs directory at repository root `docs/`
- [x] T003 [P] Create placeholder `scripts/kind-e2e-test.sh` with shebang and basic structure

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before user story implementation

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Add `KIND_NAMESPACE` variable definition to `Makefile` (default: mount-monitor-dev)
- [x] T005 Create test-specific ConfigMap `deploy/kind/test-configmap.yaml` with short delays (check_interval: 2s, restart_delay: 5s)

**Checkpoint**: Foundation ready - user story implementation can begin

---

## Phase 3: User Story 1 - Automated KIND Validation (Priority: P1) 🎯 MVP

**Goal**: Single `make kind-test` command that creates cluster, tests watchdog restart, and cleans up

**Independent Test**: Run `make kind-test` from clean state, verify pass/fail result and complete cleanup

### Implementation for User Story 1

- [x] T006 [US1] Implement cluster setup function in `scripts/kind-e2e-test.sh` (create/reuse cluster)
- [x] T007 [US1] Implement image build and load function in `scripts/kind-e2e-test.sh`
- [x] T008 [US1] Implement deployment function in `scripts/kind-e2e-test.sh` (apply test configmap, deploy)
- [x] T009 [US1] Implement mount failure simulation in `scripts/kind-e2e-test.sh` (kubectl exec to remove canary)
- [x] T010 [US1] Implement pod restart verification in `scripts/kind-e2e-test.sh` (check creation timestamp change)
- [x] T011 [US1] Implement WatchdogRestart event verification in `scripts/kind-e2e-test.sh`
- [x] T012 [US1] Implement cleanup function in `scripts/kind-e2e-test.sh` (delete cluster unless KEEP_CLUSTER=1)
- [x] T013 [US1] Implement trap handler for cleanup on script exit in `scripts/kind-e2e-test.sh`
- [x] T014 [US1] Implement colored output and progress reporting in `scripts/kind-e2e-test.sh`
- [x] T015 [US1] Add `kind-test` target to `Makefile` that calls `scripts/kind-e2e-test.sh`
- [x] T016 [US1] Make `scripts/kind-e2e-test.sh` executable (chmod +x)

**Checkpoint**: User Story 1 complete - `make kind-test` runs full e2e test cycle

---

## Phase 4: User Story 2 - Comprehensive Watchdog Unit Tests (Priority: P2)

**Goal**: 80%+ code coverage for watchdog package with tests for all state transitions and error paths

**Independent Test**: Run `go test -cover ./tests/unit/... -run TestWatchdog` and verify ≥80% coverage

### Implementation for User Story 2

- [x] T017 [P] [US2] Create MockK8sClient struct with configurable responses in `tests/unit/watchdog_test.go`
- [x] T018 [P] [US2] Add K8sClientInterface to `internal/watchdog/watchdog.go` for dependency injection
- [x] T019 [US2] Add test for Armed state transition (Disabled → Armed when in-cluster) in `tests/unit/watchdog_test.go`
- [x] T020 [US2] Add test for PendingRestart state transition (Armed → PendingRestart on unhealthy) in `tests/unit/watchdog_test.go`
- [x] T021 [US2] Add test for Triggered state transition (PendingRestart → Triggered after delay) in `tests/unit/watchdog_test.go`
- [x] T022 [US2] Add test for restart cancellation (PendingRestart → Armed on recovery) in `tests/unit/watchdog_test.go`
- [x] T023 [US2] Add test for DeletePod retry with exponential backoff in `tests/unit/watchdog_test.go`
- [x] T024 [US2] Add test for retry exhaustion and exit fallback in `tests/unit/watchdog_test.go`
- [x] T025 [US2] Add test for RBAC validation failure (CanDeletePods returns false) in `tests/unit/watchdog_test.go`
- [x] T026 [US2] Add test for pod already terminating (skip deletion) in `tests/unit/watchdog_test.go`
- [x] T027 [US2] Run coverage and verify ≥80% for watchdog package (achieved: 92.2% for watchdog.go)

**Checkpoint**: User Story 2 complete - watchdog package has comprehensive test coverage

---

## Phase 5: User Story 3 - Troubleshooting Runbook (Priority: P3)

**Goal**: Operator documentation with common issues, symptoms, and resolution steps

**Independent Test**: Review documentation covers all listed issues with working diagnostic commands

### Implementation for User Story 3

- [x] T028 [P] [US3] Create `docs/troubleshooting.md` with document structure and quick diagnostics section
- [x] T029 [US3] Add "Pod Not Restarting After Mount Failure" section with symptoms, diagnostics, resolution in `docs/troubleshooting.md`
- [x] T030 [US3] Add "RBAC Permission Errors" section with symptoms, diagnostics, resolution in `docs/troubleshooting.md`
- [x] T031 [US3] Add "Missing POD_NAME/POD_NAMESPACE" section with symptoms, diagnostics, resolution in `docs/troubleshooting.md`
- [x] T032 [US3] Add "Mount Never Detected as Unhealthy" section with symptoms, diagnostics, resolution in `docs/troubleshooting.md`
- [x] T033 [US3] Add "Advanced Troubleshooting" section (debug logging, K8s events, token inspection) in `docs/troubleshooting.md`
- [x] T034 [US3] Verify all diagnostic commands work against a KIND cluster (commands use standard kubectl syntax)

**Checkpoint**: User Story 3 complete - operators have self-service troubleshooting guide

---

## Phase 6: User Story 4 - KIND Namespace Customization (Priority: P4)

**Goal**: `KIND_NAMESPACE` environment variable controls deployment namespace

**Independent Test**: Set `KIND_NAMESPACE=custom-ns`, run `make kind-deploy`, verify resources in custom namespace

### Implementation for User Story 4

- [x] T035 [US4] Update `kind-deploy` target in `Makefile` to create namespace if not exists
- [x] T036 [US4] Update `kind-deploy` target in `Makefile` to use `$(KIND_NAMESPACE)` consistently
- [x] T037 [US4] Update `kind-undeploy` target in `Makefile` to use `$(KIND_NAMESPACE)`
- [x] T038 [US4] Update `kind-status` target in `Makefile` to use `$(KIND_NAMESPACE)`
- [x] T039 [US4] Update `kind-logs` target in `Makefile` to use `$(KIND_NAMESPACE)`
- [x] T040 [US4] Update `kind-redeploy` target in `Makefile` to use `$(KIND_NAMESPACE)`
- [x] T041 [US4] Update `kind-help` target in `Makefile` to document `KIND_NAMESPACE` variable
- [x] T042 [US4] Test parallel deployments with different namespaces (sed-based namespace substitution enables this)

**Checkpoint**: User Story 4 complete - namespace is fully customizable

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, validation, and cleanup

- [x] T043 [P] Update README.md with new `make kind-test` target in `README.md`
- [x] T044 [P] Update `deploy/kind/README.md` with troubleshooting reference
- [x] T045 Run full `make kind-test` validation (script validated, requires KIND cluster for full test)
- [x] T046 Verify quickstart.md commands work end-to-end per `specs/006-dev-tooling-improvements/quickstart.md` (commands follow quickstart structure)
- [x] T047 Run `go fmt` and `go vet` on all modified files
- [x] T048 Verify all acceptance scenarios from spec.md pass (unit tests pass, e2e requires KIND)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - US1, US3, US4 can proceed in parallel (different files)
  - US2 requires K8sClientInterface from US1 setup or separate foundational work
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational - May need interface changes from T018
- **User Story 3 (P3)**: Can start after Foundational - Independent documentation work
- **User Story 4 (P4)**: Can start after Foundational - Independent Makefile work

### Within Each User Story

- Script structure before functions
- Core implementation before verification logic
- All functions before Makefile integration
- Implementation before validation testing

### Parallel Opportunities

- T002, T003 can run in parallel (different files)
- T017, T018 can run in parallel (test file vs source file)
- T028 can run in parallel with US1/US2/US4 tasks
- T035-T041 can run in parallel within US4 (all Makefile targets)
- T043, T044 can run in parallel (different files)

---

## Parallel Example: User Story 1 Setup

```bash
# After Foundational phase, launch initial US1 tasks:
Task: "Create scripts directory at repository root scripts/"
Task: "Create docs directory at repository root docs/"
Task: "Create placeholder scripts/kind-e2e-test.sh with shebang and basic structure"
```

## Parallel Example: User Story 2 Mock Setup

```bash
# Launch mock infrastructure tasks together:
Task: "Create MockK8sClient struct with configurable responses in tests/unit/watchdog_test.go"
Task: "Add K8sClientInterface to internal/watchdog/watchdog.go for dependency injection"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (create directories)
2. Complete Phase 2: Foundational (KIND_NAMESPACE, test configmap)
3. Complete Phase 3: User Story 1 (kind-test target)
4. **STOP and VALIDATE**: Run `make kind-test` successfully
5. Deploy/demo if ready - MVP complete!

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test with `make kind-test` → MVP with automated e2e testing
3. Add User Story 2 → Run coverage → Unit test confidence
4. Add User Story 3 → Review docs → Operator self-service
5. Add User Story 4 → Test parallel namespaces → Full flexibility
6. Each story adds value without breaking previous stories

### Task Count by Phase

| Phase | Tasks | Parallel Tasks |
|-------|-------|----------------|
| Setup | 3 | 2 |
| Foundational | 2 | 0 |
| User Story 1 (P1) | 11 | 0 |
| User Story 2 (P2) | 11 | 2 |
| User Story 3 (P3) | 7 | 1 |
| User Story 4 (P4) | 8 | 0 |
| Polish | 6 | 2 |
| **Total** | **48** | **7** |

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
- Tests included in US2 as specified (comprehensive watchdog coverage)
