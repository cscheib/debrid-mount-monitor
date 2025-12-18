# Tasks: Init Container Mode

**Input**: Design documents from `/specs/010-init-container-mode/`
**Prerequisites**: plan.md (required), spec.md (required), research.md

**Tests**: Test tasks included as specified in plan.md Test Plan section.

**Organization**: This feature has tightly coupled user stories (US1: failure path, US2: success path, US3: logging) that are implemented together in a single cohesive change.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md structure:
- **Source**: `cmd/mount-monitor/`, `internal/config/`
- **Tests**: `internal/config/` (alongside source per Go convention)

---

## Phase 1: Setup

**Purpose**: No project initialization needed - this extends an existing codebase

> This phase is intentionally empty. The project structure already exists.

**Checkpoint**: Ready to proceed to implementation.

---

## Phase 2: Foundational (Config Layer Changes)

**Purpose**: Add `--init-container-mode` flag and validation adjustments to the config package. This MUST be complete before main.go changes.

**⚠️ CRITICAL**: main.go depends on config changes being complete first.

- [x] T001 Add `InitContainerMode bool` field to Config struct in internal/config/config.go
- [x] T002 Add `--init-container-mode` boolean flag parsing in Load() function in internal/config/config.go
- [x] T003 Modify Validate() to skip irrelevant validations (HTTPPort, CheckInterval, ShutdownTimeout, Watchdog) when InitContainerMode is true in internal/config/config.go

**Checkpoint**: Config package now supports init-container mode flag and adjusted validation.

---

## Phase 3: User Stories 1, 2, 3 - Init Mode Implementation (Priority: P1/P2) 🎯 MVP

**Goal**: Implement the `runInitMode()` function that checks all mounts once and exits with appropriate code.

**Why combined**: US1 (failure path), US2 (success path), and US3 (diagnostic logging) are inseparable—they're different execution paths through the same function. You cannot implement one without the others.

**Independent Test**:
- Run `./mount-monitor -c config.json --init-container-mode` against healthy mounts → exit 0
- Run against unhealthy mounts → exit 1
- Verify log output includes mount names, paths, and error details

### Tests for Init Mode

- [x] T004 [P] [US1] Add TestLoad_InitContainerModeFlag to verify flag parsing in internal/config/config_test.go
- [x] T005 [P] [US1] Add TestValidate_InitContainerMode_SkipsIrrelevant to verify validation adjustments in internal/config/config_test.go

### Implementation

- [x] T006 [US1+US2+US3] Add runInitMode(cfg *config.Config, logger *slog.Logger) int function in cmd/mount-monitor/main.go
- [x] T007 [US1+US2+US3] Add init mode early exit check after logger setup in main() in cmd/mount-monitor/main.go
- [x] T008 [US3] Add startup log message when entering init-container mode in cmd/mount-monitor/main.go

**Checkpoint**: All three user stories are complete. Init container mode is fully functional.

---

## Phase 4: Polish & Validation

**Purpose**: Final verification and cleanup

- [x] T009 Run `go build ./...` to verify compilation succeeds
- [x] T010 Run `go test ./...` to verify all tests pass
- [x] T011 Manual integration test: create temp directory with canary file, run init mode, verify exit 0
- [x] T012 Manual integration test: run init mode against non-existent path, verify exit 1 and error log
- [x] T013 Verify quickstart.md examples work as documented in specs/010-init-container-mode/quickstart.md

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: Empty - no work needed
- **Phase 2 (Foundational)**: Config changes must complete before Phase 3
- **Phase 3 (User Stories)**: Depends on Phase 2 completion
- **Phase 4 (Polish)**: Depends on Phase 3 completion

### Task Dependencies

```
T001 ──┬──→ T003 ──→ T004, T005 (tests can run after validation logic exists)
T002 ──┘

T001, T002, T003 ──→ T006 ──→ T007 ──→ T008

T006, T007, T008 ──→ T009, T010, T011, T012, T013
```

### Parallel Opportunities

**Within Phase 2:**
- T001 and T002 can run in parallel (different parts of config.go, no overlap)

**Within Phase 3 Tests:**
- T004 and T005 can run in parallel (different test functions)

**Within Phase 4:**
- T011 and T012 can run in parallel (independent manual tests)

---

## Parallel Example: Phase 2

```bash
# Launch config struct and flag changes together:
Task: "Add InitContainerMode bool field to Config struct in internal/config/config.go"
Task: "Add --init-container-mode boolean flag parsing in Load() function in internal/config/config.go"
```

---

## Implementation Strategy

### Single Increment Delivery

This feature is small enough to deliver as a single increment:

1. Complete Phase 2: Config changes (T001-T003)
2. Complete Phase 3: Main implementation (T004-T008)
3. Complete Phase 4: Validation (T009-T013)
4. **DONE**: Feature ready for PR

### Estimated Effort

- **Total tasks**: 13
- **Phase 2**: 3 tasks (config changes)
- **Phase 3**: 5 tasks (implementation + tests)
- **Phase 4**: 5 tasks (validation)

### Critical Path

T001/T002 → T003 → T006 → T007 → T008 → T009/T010

---

## Notes

- [P] tasks = different files or non-overlapping file sections, no dependencies
- [US1+US2+US3] label indicates tasks that implement multiple stories together
- All user stories share the same implementation because they're different execution paths
- Tests are included per plan.md Test Plan section
- Manual integration tests verify end-to-end behavior
- Commit after each phase for clean git history
