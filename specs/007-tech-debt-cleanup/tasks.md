# Tasks: Tech Debt Cleanup

**Input**: Design documents from `/specs/007-tech-debt-cleanup/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, quickstart.md

**Tests**: Existing tests will be updated as part of the refactoring. No new test tasks since this modifies existing test coverage.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md structure:
- **Source**: `cmd/`, `internal/` at repository root
- **Tests**: `tests/unit/` at repository root
- **Docs**: `README.md` at repository root

---

## Phase 1: Setup

**Purpose**: Prepare branch and verify starting state

- [x] T001 Verify on branch `007-tech-debt-cleanup` and all tests pass with `go test ./...`
- [x] T002 Run `go build ./...` to confirm clean build state before changes

**Checkpoint**: Starting state verified - ready to begin changes

---

## Phase 2: User Story 1 - Correct Module Namespace (Priority: P1) 🎯 MVP

**Goal**: Update Go module namespace from `github.com/chris/debrid-mount-monitor` to `github.com/cscheib/debrid-mount-monitor`

**Independent Test**: `go build ./...` succeeds and all imports resolve correctly

### Implementation for User Story 1

- [x] T003 [US1] Update module path in go.mod using `go mod edit -module github.com/cscheib/debrid-mount-monitor`
- [x] T004 [US1] Update import statements in cmd/mount-monitor/main.go
- [x] T005 [P] [US1] Update import statements in internal/config/config.go (N/A - no cross-package imports)
- [x] T006 [P] [US1] Update import statements in internal/config/file.go (N/A - no cross-package imports)
- [x] T007 [P] [US1] Update import statements in internal/monitor/monitor.go
- [x] T008 [P] [US1] Update import statements in internal/server/server.go
- [x] T009 [P] [US1] Update import statements in tests/unit/checker_test.go
- [x] T010 [P] [US1] Update import statements in tests/unit/config_file_test.go
- [x] T011 [P] [US1] Update import statements in tests/unit/config_test.go
- [x] T012 [P] [US1] Update import statements in tests/unit/monitor_test.go
- [x] T013 [P] [US1] Update import statements in tests/unit/server_test.go
- [x] T014 [P] [US1] Update import statements in tests/unit/shutdown_test.go
- [x] T015 [P] [US1] Update import statements in tests/unit/state_test.go
- [x] T016 [P] [US1] Update import statements in tests/unit/watchdog_test.go
- [x] T017 [US1] Verify with `go build ./...` - build must succeed
- [x] T018 [US1] Verify with `go test ./...` - all tests must pass
- [x] T019 [US1] Verify with `go mod tidy` - no changes should be required

**Checkpoint**: US1 complete - Module namespace corrected, all builds and tests pass

---

## Phase 3: User Story 2 - Consistent Failure Threshold Terminology (Priority: P2)

**Goal**: Replace "debounce" terminology with "failure threshold" throughout codebase

**Independent Test**: No occurrences of "debounce" in config keys, CLI flags, or log messages

### Implementation for User Story 2

**Config Layer:**
- [x] T020 [US2] Rename `DebounceThreshold` to `FailureThreshold` in Config struct in internal/config/config.go
- [x] T021 [US2] Update CLI flag from `--debounce-threshold` to `--failure-threshold` in internal/config/config.go
- [x] T022 [US2] Update default value assignment to use new field name in internal/config/config.go
- [x] T023 [US2] Update validation error message from "debounce threshold" to "failure threshold" in internal/config/config.go
- [x] T024 [P] [US2] Rename `DebounceThreshold` to `FailureThreshold` in fileConfig struct and JSON tag in internal/config/file.go

**Health/State Layer:**
- [x] T025 [P] [US2] Update function parameter name from `debounceThreshold` to `failureThreshold` in internal/health/state.go
- [x] T026 [P] [US2] Update comments referencing "debounce" to "failure threshold" in internal/health/state.go

**Monitor Layer:**
- [x] T027 [P] [US2] Rename `debounceThreshold` variable to `failureThreshold` in internal/monitor/monitor.go

**Server Layer:**
- [x] T028 [P] [US2] Update comments referencing "debounce" to "failure threshold" in internal/server/server.go

**Main Application:**
- [x] T029 [US2] Update log field name from `debounce_threshold` to `failure_threshold` in cmd/mount-monitor/main.go

**Test Files:**
- [x] T030 [P] [US2] Update terminology in tests/unit/config_test.go (variable names, assertions)
- [x] T031 [P] [US2] Update terminology in tests/unit/config_file_test.go (JSON keys, assertions)
- [x] T032 [P] [US2] Update terminology in tests/unit/state_test.go (variable names, comments)
- [x] T033 [P] [US2] Update terminology in tests/unit/server_test.go (variable names)
- [x] T034 [P] [US2] Update terminology in tests/unit/monitor_test.go (variable names)

**Verification:**
- [x] T035 [US2] Verify with `go build ./...` - build must succeed
- [x] T036 [US2] Verify with `go test ./...` - all tests must pass
- [x] T037 [US2] Verify no "debounce" references in .go files using `grep -ri "debounce" --include="*.go" .`

**Checkpoint**: US2 complete - Terminology consistently uses "failure threshold"

---

## Phase 4: User Story 3 - Clear Path Configuration Documentation (Priority: P3)

**Goal**: Document that mount paths can be relative or absolute, canary files are relative to mount

**Independent Test**: README clearly states path configuration rules

### Implementation for User Story 3

- [x] T038 [US3] Add "Path Configuration" subsection to JSON Configuration section in README.md explaining relative/absolute mount paths
- [x] T039 [US3] Update JSON configuration example in README.md to show both relative and absolute path examples
- [x] T040 [US3] Add note clarifying canary file paths are always relative to mount path in README.md
- [x] T041 [US3] Update inline comments in internal/config/config.go for Path field to mention relative/absolute support

**Checkpoint**: US3 complete - Path configuration is clearly documented

---

## Phase 5: User Story 4 - Simplified Configuration (Priority: P4)

**Goal**: Remove environment variable configuration support, keep only JSON config and CLI flags

**Independent Test**: Setting config env vars has no effect; only K8s runtime vars remain

### Implementation for User Story 4

**Remove Environment Variable Processing:**
- [x] T042 [US4] Remove `MOUNT_PATHS` environment variable handling in internal/config/config.go
- [x] T043 [US4] Remove `CANARY_FILE` environment variable handling in internal/config/config.go
- [x] T044 [US4] Remove `CHECK_INTERVAL` environment variable handling in internal/config/config.go
- [x] T045 [US4] Remove `READ_TIMEOUT` environment variable handling in internal/config/config.go
- [x] T046 [US4] Remove `SHUTDOWN_TIMEOUT` environment variable handling in internal/config/config.go
- [x] T047 [US4] Remove `DEBOUNCE_THRESHOLD`/`FAILURE_THRESHOLD` environment variable handling in internal/config/config.go
- [x] T048 [US4] Remove `HTTP_PORT` environment variable handling in internal/config/config.go
- [x] T049 [US4] Remove `LOG_LEVEL` environment variable handling in internal/config/config.go
- [x] T050 [US4] Remove `LOG_FORMAT` environment variable handling in internal/config/config.go
- [x] T051 [US4] Remove `WATCHDOG_ENABLED` environment variable handling in internal/config/config.go
- [x] T052 [US4] Remove `WATCHDOG_RESTART_DELAY` environment variable handling in internal/config/config.go
- [x] T053 [US4] Remove `os` import if no longer needed in internal/config/config.go

**Update Tests:**
- [x] T054 [US4] Remove environment variable tests in tests/unit/config_file_test.go

**Documentation Updates:**
- [x] T055 [US4] Remove "Environment Variables" section from README.md
- [x] T056 [US4] Update Configuration precedence description in README.md (remove env vars from chain)
- [x] T057 [US4] Remove env var references from Docker example in README.md
- [x] T058 [US4] Remove env var references from Kubernetes example in README.md

**Verification:**
- [x] T059 [US4] Verify K8s runtime vars still present in internal/watchdog/k8s_client.go (KUBERNETES_SERVICE_HOST, KUBERNETES_SERVICE_PORT)
- [x] T060 [US4] Verify K8s runtime vars still present in cmd/mount-monitor/main.go (POD_NAME, POD_NAMESPACE)
- [x] T061 [US4] Verify with `go build ./...` - build must succeed
- [x] T062 [US4] Verify with `go test ./...` - all tests must pass
- [x] T063 [US4] Verify config env vars removed using `grep -r "os.Getenv" internal/config/`

**Checkpoint**: US4 complete - Environment variable configuration removed, K8s runtime vars preserved

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification, documentation, and breaking change notice

- [x] T064 Update README.md terminology section if needed (align with "failure threshold")
- [x] T065 Final verification: `go build ./...` succeeds
- [x] T066 Final verification: `go test ./...` passes all tests
- [x] T067 Final verification: `go mod tidy` makes no changes
- [x] T068 Run quickstart.md verification checklist
- [x] T069 [P] Add breaking change notice to CHANGELOG.md or create release notes (created BREAKING_CHANGES.md)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - verify starting state
- **US1 (Phase 2)**: Depends on Setup - MUST complete before US2-US4 (changes imports)
- **US2 (Phase 3)**: Depends on US1 - builds on working namespace
- **US3 (Phase 4)**: Independent of US2 - documentation only (can run parallel to US2)
- **US4 (Phase 5)**: Independent of US2/US3 - can run after US1 (can run parallel to US2/US3)
- **Polish (Phase 6)**: Depends on all user stories complete

### User Story Dependencies

```
Setup → US1 (namespace) → US2 (terminology) → Polish
                       ↘ US3 (path docs)    ↗
                       ↘ US4 (env vars)     ↗
```

- **US1 (P1)**: MUST complete first - affects all imports
- **US2 (P2)**: After US1 - builds on correct namespace
- **US3 (P3)**: After US1 - can parallel with US2, US4 (only touches README)
- **US4 (P4)**: After US1 - can parallel with US2, US3 (different code sections)

### Within Each User Story

- Config changes before dependent modules
- Code changes before test updates
- Verification steps at end of each story

### Parallel Opportunities

**Within US1** (after T003):
- T005-T016 can all run in parallel (different files)

**Within US2**:
- T024-T028 can run in parallel (different files)
- T030-T034 can run in parallel (different test files)

**After US1 completes**:
- US2, US3, and US4 can run in parallel (different concerns)

---

## Parallel Example: User Story 1 Imports

```bash
# After T003 (go.mod update), launch all import updates together:
Task: "Update import statements in internal/config/config.go"
Task: "Update import statements in internal/config/file.go"
Task: "Update import statements in internal/monitor/monitor.go"
Task: "Update import statements in internal/server/server.go"
Task: "Update import statements in tests/unit/checker_test.go"
# ... all other import updates
```

## Parallel Example: After US1 Completes

```bash
# Three developers can work in parallel:
Developer A: US2 (terminology alignment)
Developer B: US3 (path documentation)
Developer C: US4 (environment variable removal)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: US1 (namespace)
3. **STOP and VALIDATE**: Build and tests pass, namespace correct
4. This alone fixes the module/GitHub mismatch

### Incremental Delivery

1. US1 → Namespace fixed (critical functionality)
2. US2 → Terminology aligned (improved UX)
3. US3 → Path docs added (better documentation)
4. US4 → Env vars removed (simplified configuration, breaking change)
5. Each story adds value independently

### Recommended Order for Single Developer

1. Setup (T001-T002)
2. US1 completely (T003-T019) - verify before continuing
3. US2 completely (T020-T037) - verify before continuing
4. US3 completely (T038-T041) - quick, docs only
5. US4 completely (T042-T063) - larger, breaking change
6. Polish (T064-T069) - final verification

---

## Notes

- [P] tasks = different files, no dependencies within that phase
- [Story] label maps task to specific user story for traceability
- US1 MUST complete before US2-US4 due to import dependencies
- US3 and US4 are independent of US2 and each other
- Verify tests pass after each user story before proceeding
- Breaking change (US4) should be clearly communicated in release notes
