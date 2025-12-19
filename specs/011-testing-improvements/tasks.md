# Tasks: Testing Quality Improvements

**Input**: Design documents from `/specs/011-testing-improvements/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, quickstart.md

**Tests**: This feature IS about testing infrastructure. All tasks involve test code or test tooling.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Single Go project: `internal/`, `cmd/`, `tests/` at repository root

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add goleak dependency and configure test tooling

- [x] T001 Add goleak dependency: `go get -t go.uber.org/goleak` and verify zero transitive deps in go.mod
- [x] T002 [P] Add `test-race` target to Makefile: `go test -race ./...`
- [x] T003 [P] Add `test-cover` target to Makefile: generate coverage.out and coverage.html
- [x] T004 [P] Add `test-all` target to Makefile: combined race + coverage
- [x] T005 [P] Update .github/workflows/ci.yml to upload coverage artifact

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Create shared test utilities that ALL user stories depend on

**⚠️ CRITICAL**: User story implementation requires these test helpers

- [x] T006 Create internal/testutil/ directory structure
- [x] T007 [P] Implement testutil.Logger() in internal/testutil/logger.go - returns silent slog.Logger for tests
- [x] T008 [P] Implement testutil.TempConfig() in internal/testutil/config.go - creates temp config file, returns path
- [x] T009 [P] Implement testutil.PollUntil() in internal/testutil/poll.go - polls condition with timeout

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 + 2 - Goroutine Leak & Race Detection (Priority: P1) 🎯 MVP

**Goal**: Enable detection of goroutine leaks and race conditions in the test suite

**Independent Test**: Run `make test-race` and verify zero races; run tests with goleak and verify zero leaks

### US1: Goroutine Leak Detection

- [x] T010 [US1] Add goleak.VerifyNone(t) to internal/watchdog/watchdog_test.go for goroutine-heavy tests
- [x] T011 [P] [US1] Add goleak.VerifyNone(t) to internal/monitor/monitor_test.go for mount check goroutines
- [x] T012 [P] [US1] Add goleak.VerifyNone(t) to internal/server/server_test.go for HTTP handler tests
- [x] T013 [US1] Verify existing tests pass with goleak enabled (fix any pre-existing leaks)

### US2: Race Condition Detection

- [x] T014 [US2] Add concurrent access test for health.State in internal/health/state_test.go - multiple goroutines read/write
- [x] T015 [P] [US2] Add concurrent mount state change test in internal/watchdog/watchdog_test.go
- [x] T016 [US2] Run full test suite with `make test-race` and verify zero race conditions detected

**Checkpoint**: Goroutine leaks and race conditions are now detectable. MVP complete.

---

## Phase 4: User Story 3 - Coverage Tracking (Priority: P2)

**Goal**: Developers can generate and view coverage reports locally and in CI

**Independent Test**: Run `make test-cover` and verify coverage.html is generated with per-package breakdown

- [ ] T017 [US3] Verify `make test-cover` generates coverage.html with per-package percentages
- [ ] T018 [US3] Verify CI workflow uploads coverage artifact on pull requests
- [ ] T019 [US3] Document coverage commands in quickstart.md (already created in specs/)

**Checkpoint**: Coverage reporting is available locally and in CI

---

## Phase 5: User Story 4 - Watchdog Test Coverage (Priority: P2)

**Goal**: Expand watchdog tests from 45% to 80%+ coverage

**Independent Test**: Run `go test -coverprofile=c.out ./internal/watchdog/... && go tool cover -func=c.out` and verify 80%+

- [ ] T020 [US4] Add timer cancellation test in internal/watchdog/watchdog_test.go - mount recovers during pending restart
- [ ] T021 [P] [US4] Add API retry exhaustion test in internal/watchdog/watchdog_test.go - retries fail, process exits
- [ ] T022 [P] [US4] Add pod-already-terminating test in internal/watchdog/watchdog_test.go - pod terminating check
- [ ] T023 [US4] Add all state machine transition tests in internal/watchdog/watchdog_test.go - Disabled→Armed→PendingRestart→Triggered
- [ ] T024 [US4] Verify watchdog coverage reaches 80%+ target

**Checkpoint**: Watchdog component has comprehensive test coverage

---

## Phase 6: User Story 5 - Entry Point Tests (Priority: P2)

**Goal**: Add tests for main.go (currently 0% → target 60%+)

**Independent Test**: Run `go test -coverprofile=c.out ./cmd/... && go tool cover -func=c.out` and verify 60%+

- [x] T025 [US5] Refactor cmd/mount-monitor/main.go to extract setupLogger() as testable function
- [x] T026 [US5] Refactor cmd/mount-monitor/main.go to extract runInitMode() as testable function
- [x] T027 [US5] Create cmd/mount-monitor/main_test.go with test for setupLogger() - verify stdout/stderr routing
- [x] T028 [P] [US5] Add test for runInitMode() in cmd/mount-monitor/main_test.go - temp mounts, exit codes
- [ ] T029 [P] [US5] Add test for shutdown coordination in cmd/mount-monitor/main_test.go - SIGTERM handling
- [ ] T030 [US5] Verify main.go coverage reaches 60%+ target

**Checkpoint**: Entry point logic is now tested

---

## Phase 7: User Story 6 - E2E Tests (Priority: P3)

**Goal**: Expand KIND E2E tests for mount failure and recovery scenarios

**Independent Test**: Run `make kind-test` and verify mount failure detection, recovery, and pod restart scenarios pass

- [ ] T031 [US6] Create tests/e2e/ directory structure
- [ ] T032 [US6] Create mount failure detection test in tests/e2e/mount_failure_test.go - remove canary, verify unhealthy
- [ ] T033 [P] [US6] Add recovery cancellation test in tests/e2e/mount_failure_test.go - restore canary during pending restart
- [ ] T034 [P] [US6] Add pod restart verification test in tests/e2e/mount_failure_test.go - pod deleted, event created
- [ ] T035 [US6] Enhance scripts/kind-e2e-test.sh with recovery verification steps
- [ ] T036 [US6] Verify `make kind-test` passes all 3 scenarios

**Checkpoint**: E2E tests verify complete system behavior in KIND

---

## Phase 8: User Story 7 - Shared Test Utilities Adoption (Priority: P3)

**Goal**: Refactor existing tests to use testutil helpers (reduces duplication)

**Independent Test**: Verify all tests still pass after refactoring to use testutil

- [ ] T037 [US7] Refactor internal/watchdog/watchdog_test.go to use testutil.Logger()
- [ ] T038 [P] [US7] Refactor internal/server/server_test.go to use testutil.Logger()
- [ ] T039 [P] [US7] Refactor internal/monitor/monitor_test.go to use testutil.Logger() and testutil.PollUntil()
- [ ] T040 [P] [US7] Refactor internal/config/file_test.go to use testutil.TempConfig()
- [ ] T041 [US7] Verify all tests pass after testutil adoption

**Checkpoint**: Test utilities are adopted across the codebase

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and documentation

- [ ] T042 Run full test suite with race detection: `make test-race`
- [ ] T043 Run full test suite with coverage: `make test-cover`
- [ ] T044 Verify overall coverage reaches 75%+ target
- [ ] T045 Verify all success criteria from spec.md are met
- [ ] T046 Update CLAUDE.md with goleak in approved dependencies (already done by script)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on T001 (goleak) from Setup
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - US1 + US2 (Phase 3): Can proceed after Foundational
  - US3-7 (Phase 4-8): Can proceed after Foundational, can run in parallel with each other
- **Polish (Phase 9)**: Depends on all user stories being complete

### User Story Dependencies

| Story | Depends On | Can Parallel With |
|-------|------------|-------------------|
| US1 (Goroutine Leaks) | Foundational | US2 |
| US2 (Race Detection) | Foundational | US1 |
| US3 (Coverage) | Foundational | US4, US5, US6, US7 |
| US4 (Watchdog Tests) | Foundational | US3, US5, US6, US7 |
| US5 (main.go Tests) | Foundational | US3, US4, US6, US7 |
| US6 (E2E Tests) | Foundational | US3, US4, US5, US7 |
| US7 (testutil Adoption) | Foundational (T006-T009) | US3, US4, US5, US6 |

### Within Each User Story

- Infrastructure tasks before test implementation
- Test implementation before verification
- Verification task marks story complete

### Parallel Opportunities

- T002, T003, T004, T005 (Setup Makefile/CI) - all [P]
- T007, T008, T009 (testutil helpers) - all [P]
- T011, T012 (goleak in monitor/server) - both [P]
- T014, T015 (concurrent tests) - both [P]
- T021, T022 (watchdog edge cases) - both [P]
- T028, T029 (main.go tests) - both [P]
- T033, T034 (E2E scenarios) - both [P]
- T037, T038, T039, T040 (testutil adoption) - all [P] except T037

---

## Parallel Example: Phase 2 (Foundational)

```bash
# Launch all testutil helpers together:
Task: "Implement testutil.Logger() in internal/testutil/logger.go"
Task: "Implement testutil.TempConfig() in internal/testutil/config.go"
Task: "Implement testutil.PollUntil() in internal/testutil/poll.go"
```

## Parallel Example: Phase 3 (US1 + US2)

```bash
# US1 tasks can run in parallel with US2 tasks:
# US1:
Task: "Add goleak.VerifyNone(t) to internal/monitor/monitor_test.go"
Task: "Add goleak.VerifyNone(t) to internal/server/server_test.go"
# US2:
Task: "Add concurrent access test for health.State in internal/health/state_test.go"
```

---

## Implementation Strategy

### MVP First (Phase 1-3 Only)

1. Complete Phase 1: Setup (goleak, Makefile, CI)
2. Complete Phase 2: Foundational (testutil package)
3. Complete Phase 3: US1 + US2 (leak detection + race detection)
4. **STOP and VALIDATE**: Run `make test-race` and verify zero issues
5. **MVP COMPLETE**: Core testing quality improvements are in place

### Incremental Delivery

1. Phase 1-3 → MVP (leak + race detection)
2. Add Phase 4 → Coverage tracking visible
3. Add Phase 5 → Watchdog fully tested
4. Add Phase 6 → main.go tested
5. Add Phase 7 → E2E scenarios expanded
6. Add Phase 8 → Tests use shared utilities
7. Phase 9 → Final validation

### Parallel Team Strategy

With multiple developers after Foundational is complete:

- Developer A: US1 (goleak) + US4 (watchdog tests)
- Developer B: US2 (race tests) + US5 (main.go tests)
- Developer C: US6 (E2E tests) + US7 (testutil adoption)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Run `make test-race` frequently during development
