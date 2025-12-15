# Tasks: JSON Configuration File

**Input**: Design documents from `/specs/002-json-config/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are included for this feature to ensure backwards compatibility and validate all acceptance scenarios.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Project structure**: `cmd/`, `internal/`, `tests/` at repository root
- Based on existing Go project structure from plan.md

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Foundation for JSON config file support

- [ ] T001 Add `--config` / `-c` CLI flag definition in internal/config/config.go
- [ ] T002 [P] Create internal/config/file.go skeleton with FileConfig struct (JSON-specific, references MountConfig from config.go)
- [ ] T003 [P] Add Duration type with custom UnmarshalJSON in internal/config/file.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T004 Add MountConfig struct to internal/config/config.go with Name, Path, CanaryFile, FailureThreshold fields (canonical location, used by file.go and main.go)
- [ ] T005 Add ConfigFile field to Config struct in internal/config/config.go
- [ ] T006 Add Mounts []MountConfig field to Config struct in internal/config/config.go
- [ ] T007 Add Name and FailureThreshold fields to Mount struct in internal/health/state.go
- [ ] T008 Update NewMount constructor signature in internal/health/state.go to accept (name, path, canaryFile string, failureThreshold int)
- [ ] T009 Update all existing NewMount call sites to use new signature (cmd/mount-monitor/main.go, tests)

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Load Configuration from JSON File (Priority: P1) 🎯 MVP

**Goal**: Enable loading mount configurations from a JSON file via `--config` flag or `./config.json` default

**Independent Test**: Create a JSON config file with mount definitions, start application with `--config path/to/config.json`, verify mounts are loaded

### Tests for User Story 1

- [ ] T010 [P] [US1] Test JSON file parsing with valid config in tests/unit/config_file_test.go
- [ ] T011 [P] [US1] Test `--config` flag loads specified file in tests/unit/config_file_test.go
- [ ] T012 [P] [US1] Test `./config.json` default location discovery in tests/unit/config_file_test.go
- [ ] T013 [P] [US1] Test backwards compatibility: no config file uses env vars in tests/unit/config_test.go

### Implementation for User Story 1

- [ ] T014 [US1] Implement LoadFromFile function in internal/config/file.go (read file, parse JSON)
- [ ] T015 [US1] Implement config file discovery logic in internal/config/file.go (--config flag → ./config.json → none)
- [ ] T016 [US1] Implement FileConfig to Config mapping in internal/config/file.go
- [ ] T017 [US1] Update Load() in internal/config/config.go to call LoadFromFile after defaults, before env vars
- [ ] T018 [US1] Handle "file not found" error when --config explicitly specified in internal/config/file.go
- [ ] T019 [US1] Silently skip when ./config.json doesn't exist (backwards compatible) in internal/config/file.go

**Checkpoint**: User Story 1 complete - can load config from JSON file, falls back to env vars when absent

---

## Phase 4: User Story 2 - Per-Mount Configuration (Priority: P2)

**Goal**: Support individual canary file and failure threshold per mount

**Independent Test**: Create config with two mounts having different canary files and thresholds, verify each mount uses its specific settings

### Tests for User Story 2

- [ ] T020 [P] [US2] Test per-mount canary file override in tests/unit/config_file_test.go
- [ ] T021 [P] [US2] Test per-mount failureThreshold override in tests/unit/config_file_test.go
- [ ] T022 [P] [US2] Test default inheritance when per-mount values not specified in tests/unit/config_file_test.go
- [ ] T023 [P] [US2] Test Mount.Name field in health status in tests/unit/state_test.go

### Implementation for User Story 2

- [ ] T024 [US2] Implement per-mount default inheritance in internal/config/file.go (mount inherits global canaryFile/threshold if not specified)
- [ ] T025 [US2] Update mount creation loop in cmd/mount-monitor/main.go to use MountConfig values
- [ ] T026 [US2] Pass per-mount failureThreshold to health.NewMount in cmd/mount-monitor/main.go
- [ ] T026a [US2] Update monitor.go checkAll() to pass mount.FailureThreshold to UpdateState instead of global threshold
- [ ] T027 [US2] Update Mount.UpdateState to use per-mount FailureThreshold instead of global in internal/health/state.go
- [ ] T028 [US2] Include mount Name in MountStatusResponse in internal/server/server.go

**Checkpoint**: User Story 2 complete - each mount can have independent canary file and failure threshold

---

## Phase 5: User Story 3 - Configuration Validation on Startup (Priority: P3)

**Goal**: Validate config file at startup with clear, actionable error messages

**Independent Test**: Provide malformed JSON or invalid config values, verify specific error messages are returned

### Tests for User Story 3

- [ ] T029 [P] [US3] Test invalid JSON syntax error message in tests/unit/config_file_test.go
- [ ] T030 [P] [US3] Test missing required "path" field error in tests/unit/config_file_test.go
- [ ] T031 [P] [US3] Test invalid failureThreshold (zero/negative) error in tests/unit/config_file_test.go
- [ ] T032 [P] [US3] Test empty mounts array error in tests/unit/config_file_test.go
- [ ] T033 [P] [US3] Test verbose startup logging output in tests/unit/config_test.go

### Implementation for User Story 3

- [ ] T034 [US3] Implement ValidateFileConfig function in internal/config/file.go
- [ ] T035 [US3] Add mount-level validation: path required, failureThreshold >= 1 in internal/config/file.go
- [ ] T036 [US3] Format error messages with mount index and name (e.g., "mount[2] 'backup': ...") in internal/config/file.go
- [ ] T037 [US3] Add verbose config logging at startup in cmd/mount-monitor/main.go (config source, mount count, settings)
- [ ] T038 [US3] Log each mount's configuration at info level in cmd/mount-monitor/main.go (name, path, canaryFile, threshold)

**Checkpoint**: User Story 3 complete - all validation and logging requirements met

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final integration and documentation

- [ ] T039 [P] Update README.md with JSON config file usage examples
- [ ] T040 [P] Add example config.json to repository root or examples/ directory
- [ ] T041 Run all existing tests to verify backwards compatibility
- [ ] T042 Run quickstart.md validation scenarios manually
- [ ] T043 Verify constitution compliance (no external dependencies added)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - User stories should be completed in priority order (P1 → P2 → P3)
  - Each builds on the previous
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational - Core JSON loading
- **User Story 2 (P2)**: Depends on US1 completion - Adds per-mount customization
- **User Story 3 (P3)**: Can start after Foundational, but validation builds on US1/US2 patterns

### Within Each User Story

- Tests written first, verify they fail
- Implementation tasks in dependency order
- Story complete before moving to next priority

### Parallel Opportunities

**Phase 1 (Setup)**:
- T002 and T003 can run in parallel (different concerns in same file skeleton)

**Phase 2 (Foundational)**:
- Tasks are sequential due to struct dependencies

**Phase 3 (US1 Tests)**:
- T010, T011, T012, T013 can all run in parallel (independent test cases)

**Phase 4 (US2 Tests)**:
- T020, T021, T022, T023 can all run in parallel
- T026a depends on T026 (same subsystem)

**Phase 5 (US3 Tests)**:
- T029, T030, T031, T032, T033 can all run in parallel

**Phase 6 (Polish)**:
- T039 and T040 can run in parallel (documentation)

---

## Parallel Example: User Story 1 Tests

```bash
# Launch all US1 tests together:
Task: "Test JSON file parsing with valid config in tests/unit/config_file_test.go"
Task: "Test --config flag loads specified file in tests/unit/config_file_test.go"
Task: "Test ./config.json default location discovery in tests/unit/config_file_test.go"
Task: "Test backwards compatibility: no config file uses env vars in tests/unit/config_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T009)
3. Complete Phase 3: User Story 1 (T010-T019)
4. **STOP and VALIDATE**: Test JSON config loading independently
5. Can deploy with basic JSON config support

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → **MVP: JSON config loading works**
3. Add User Story 2 → Test independently → **Per-mount customization works**
4. Add User Story 3 → Test independently → **Full validation and logging**
5. Polish phase → Documentation and examples

### Sequential Team Strategy

For a single developer:

1. Complete all phases in order (1 → 2 → 3 → 4 → 5 → 6)
2. Test each user story checkpoint before proceeding
3. Commit after each logical task group

---

## Notes

- [P] tasks = different files or independent test cases, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently testable at its checkpoint
- Backwards compatibility is critical - existing env var/CLI deployments must continue working
- Constitution compliance: no external dependencies (use stdlib encoding/json only)
