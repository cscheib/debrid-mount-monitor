# Tasks: GitHub Issues Batch Implementation

**Input**: Design documents from `/specs/003-github-issues-batch/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅

**Tests**: Unit tests included for file size limit (spec acceptance criteria). No TDD requested.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: `cmd/`, `internal/`, `tests/` at repository root
- This is an existing Go project with established structure

---

## Phase 1: Setup (No Changes Required)

**Purpose**: Project initialization and basic structure

This feature modifies existing files only. No new project setup required.

**Checkpoint**: ✅ Project structure already exists - proceed directly to implementation

---

## Phase 2: Foundational (No Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

This feature makes incremental improvements to an existing codebase. All dependencies are already in place:
- ✅ Go 1.21+ with slog support
- ✅ Standard library imports (runtime, log/slog)
- ✅ Existing config loading infrastructure
- ✅ Existing test framework

**Checkpoint**: ✅ Foundation ready - user story implementation can begin immediately

---

## Phase 3: User Story 1 - Security-Conscious Deployment (Priority: P1) 🎯 MVP

**Goal**: Add security hardening to config file loading (file size limit + permission warning)

**Independent Test**: Create oversized config file (>1MB) and verify rejection. Create world-writable config and verify warning log on Unix.

**Issues Addressed**: #17 (file size limit), #15 (permission warning)

### Implementation for User Story 1

- [x] T001 [P] [US1] Add imports for `runtime` and `log/slog` in internal/config/file.go
- [x] T002 [P] [US1] Define `maxConfigFileSize` constant (1MB) in internal/config/file.go
- [x] T003 [US1] Add file size check before reading config in internal/config/file.go:loadFromFile()
- [x] T004 [US1] Add world-writable permission warning with `runtime.GOOS` check (Unix only, skip on Windows per FR-003) in internal/config/file.go:loadFromFile()
- [x] T005 [P] [US1] Add unit test for file size limit rejection (>1MB) in tests/unit/config_file_test.go
- [x] T006 [P] [US1] Add unit test for normal file acceptance (<1MB) in tests/unit/config_file_test.go
- [x] T006b [P] [US1] Add boundary test for exactly 1MB file (should be accepted per edge case) in tests/unit/config_file_test.go

**Checkpoint**: Config file security hardening complete. Files >1MB rejected, world-writable files generate warning.

---

## Phase 4: User Story 2 - Operator Debugging Configuration Issues (Priority: P2)

**Goal**: Improve error messages to explain WHY constraints exist

**Independent Test**: Provide config with readTimeout >= checkInterval, verify error message explains the implication.

**Issues Addressed**: #8 (error message clarity)

### Implementation for User Story 2

- [x] T007 [US2] Update ReadTimeout validation error message in internal/config/config.go:Validate()

**Checkpoint**: Error message now explains that health checks would overlap or never complete.

---

## Phase 5: User Story 3 - Developer Understanding Code Limitations (Priority: P2)

**Goal**: Document goroutine leak limitation in health checker for maintainers

**Independent Test**: Code review confirms comprehensive documentation explaining the limitation and rationale.

**Issues Addressed**: #7 (goroutine documentation)

### Implementation for User Story 3

- [x] T008 [US3] Add comprehensive comment block explaining goroutine leak limitation in internal/health/checker.go:Check()

**Checkpoint**: Future maintainers can understand the goroutine behavior without debugging.

---

## Phase 6: User Story 4 - Performance-Sensitive Deployment (Priority: P3)

**Goal**: Eliminate per-log-call allocations in multiStreamHandler

**Independent Test**: Code review confirms handlers are pre-created at startup, not on every Handle() call.

**Issues Addressed**: #12 (handler caching)

### Implementation for User Story 4

- [x] T009 [US4] Modify multiStreamHandler struct to store cached handlers in cmd/mount-monitor/main.go
- [x] T010 [US4] Pre-create stdout/stderr handlers in setupLogger() in cmd/mount-monitor/main.go
- [x] T011 [US4] Simplify Handle() method to use cached handlers in cmd/mount-monitor/main.go

**Checkpoint**: Logging overhead reduced from O(n) allocations per log to O(1) at startup.

---

## Phase 7: Issue Closures (Parallel with Implementation)

**Purpose**: Close issues that were triaged as "won't do" with explanatory comments

- [x] T012 [P] Close issue #16 with comment explaining silent fallback is acceptable
- [x] T013 [P] Close issue #14 with comment explaining README is sufficient
- [x] T014 [P] Close issue #13 with comment explaining Name is immutable
- [x] T015 [P] Close issue #11 with comment explaining over-engineering concern
- [x] T016 [P] Close issue #10 with comment explaining out of scope
- [x] T017 [P] Close issue #9 with comment explaining internal packages don't need public docs

**Checkpoint**: All 6 "won't do" issues closed with explanatory comments.

---

## Phase 8: Polish & Verification

**Purpose**: Final validation and PR creation

- [x] T018 Run `go test ./...` to verify all tests pass
- [x] T019 Run `go vet ./...` to verify no static analysis issues
- [x] T020 Run `go build ./...` to verify successful build
- [x] T021 [P] Manual test: Create config file >1MB and verify rejection
- [x] T022 [P] Manual test: Create world-writable config on Unix and verify warning
- [x] T023 Create commit with all changes
- [x] T024 Push branch and create PR

**Checkpoint**: All implementations verified, PR ready for review.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: ✅ N/A - existing project
- **Foundational (Phase 2)**: ✅ N/A - no new infrastructure needed
- **User Stories (Phase 3-6)**: Can proceed independently - different files
- **Issue Closures (Phase 7)**: Can run in parallel with any phase
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

| Story | Files Modified | Can Parallelize With |
|-------|----------------|---------------------|
| US1 (P1) | `internal/config/file.go`, `tests/unit/config_file_test.go` | US2, US3, US4 |
| US2 (P2) | `internal/config/config.go` | US1, US3, US4 |
| US3 (P2) | `internal/health/checker.go` | US1, US2, US4 |
| US4 (P3) | `cmd/mount-monitor/main.go` | US1, US2, US3 |

**Note**: All user stories modify different files - can be implemented in any order or in parallel.

### Within Each User Story

- US1: Imports → Constant → Size check → Permission check → Tests
- US2: Single error message change (no internal dependencies)
- US3: Single comment addition (no internal dependencies)
- US4: Struct change → Handler creation → Handle() simplification

### Parallel Opportunities

```
All user stories can run in parallel (different files):
- US1: internal/config/file.go
- US2: internal/config/config.go
- US3: internal/health/checker.go
- US4: cmd/mount-monitor/main.go

All issue closures (T012-T017) can run in parallel with GitHub CLI.
```

---

## Parallel Example: Full Feature Implementation

```bash
# Launch all user stories in parallel (different files):
Task: "[US1] Add security hardening to internal/config/file.go"
Task: "[US2] Improve error message in internal/config/config.go"
Task: "[US3] Document goroutine leak in internal/health/checker.go"
Task: "[US4] Cache slog handlers in cmd/mount-monitor/main.go"

# Launch all issue closures in parallel:
Task: "Close issue #16 as won't do"
Task: "Close issue #14 as won't do"
Task: "Close issue #13 as won't do"
Task: "Close issue #11 as won't do"
Task: "Close issue #10 as won't do"
Task: "Close issue #9 as won't do"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. ✅ Setup complete (existing project)
2. ✅ Foundation complete (existing infrastructure)
3. Complete Phase 3: User Story 1 (security hardening)
4. **STOP and VALIDATE**: Test file size limit and permission warning
5. Can deploy/demo with just security improvements

### Incremental Delivery

1. US1 (Security) → Test → Value delivered
2. US2 (Error Messages) → Test → Value delivered
3. US3 (Documentation) → Review → Value delivered
4. US4 (Performance) → Review → Value delivered
5. Each story adds independent value

### Recommended Execution Order

For a single developer:
1. T001-T006 (US1 - highest priority, security)
2. T007 (US2 - quick win, 1 line change)
3. T008 (US3 - quick win, comment only)
4. T009-T011 (US4 - most complex)
5. T012-T017 (Issue closures - can be interspersed)
6. T018-T024 (Verification and PR)

---

## Task Summary

| Phase | Tasks | Parallel Opportunities |
|-------|-------|----------------------|
| Phase 1: Setup | 0 | N/A |
| Phase 2: Foundational | 0 | N/A |
| Phase 3: US1 Security | 7 | T001-T002 parallel, T005-T006-T006b parallel |
| Phase 4: US2 Error Msg | 1 | Can run with any US |
| Phase 5: US3 Docs | 1 | Can run with any US |
| Phase 6: US4 Performance | 3 | Can run with any US |
| Phase 7: Issue Closures | 6 | All parallel |
| Phase 8: Polish | 7 | T021-T022 parallel |
| **Total** | **25** | **High parallelism potential** |

---

## Notes

- All user stories modify different files - maximum parallelism
- Issue closures use `gh issue close` - can run anytime
- Tests are included for US1 only (file size limit - per spec requirements)
- All tasks are specific enough for LLM execution without additional context
- Commit after each user story for clean history
