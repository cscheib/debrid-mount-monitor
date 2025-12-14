# Tasks: Mount Health Monitor

**Input**: Design documents from `/specs/001-mount-health-monitor/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are included as they are essential for validating health monitoring behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: Go standard layout with `cmd/` and `internal/` at repository root
- Paths follow plan.md structure

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, Go module setup, and CI configuration

- [ ] T001 Create project directory structure per plan.md (cmd/, internal/, build/, .github/)
- [ ] T002 Initialize Go module with `go mod init` in repository root
- [ ] T003 [P] Create Dockerfile for multi-stage build in build/Dockerfile
- [ ] T004 [P] Create debug Dockerfile with Alpine base in build/Dockerfile.debug
- [ ] T005 [P] Create GitHub Actions CI workflow in .github/workflows/ci.yml (FR-016: build ARM64/AMD64, run tests)
- [ ] T006 [P] Create .gitignore for Go project in repository root

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T007 Implement Config struct and parsing in internal/config/config.go (FR-011: env vars + flags)
- [ ] T008 Implement config validation rules in internal/config/config.go
- [ ] T009 [P] Implement HealthStatus enum in internal/health/state.go
- [ ] T010 [P] Implement Mount struct in internal/health/state.go
- [ ] T011 Implement structured logging setup in cmd/mount-monitor/main.go (FR-012, FR-013: stdout/stderr, JSON format)
- [ ] T012 [P] Write unit tests for config parsing in tests/unit/config_test.go

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Continuous Mount Health Monitoring (Priority: P1) 🎯 MVP

**Goal**: Continuously monitor configured mount paths by reading canary files and tracking health state

**Independent Test**: Configure a mount path, start the monitor, verify it detects healthy/unhealthy states and logs transitions

### Tests for User Story 1

- [ ] T013 [P] [US1] Write unit tests for health checker in tests/unit/checker_test.go
- [ ] T014 [P] [US1] Write unit tests for state management in tests/unit/state_test.go

### Implementation for User Story 1

- [ ] T015 [US1] Implement CheckResult struct in internal/health/state.go
- [ ] T016 [US1] Implement StateTransition struct in internal/health/state.go
- [ ] T017 [US1] Implement canary file health check with timeout in internal/health/checker.go (FR-002: read canary file, 5s default timeout)
- [ ] T018 [US1] Implement debounce/threshold logic in internal/health/state.go (FR-004: 3 consecutive failures)
- [ ] T019 [US1] Implement state transition detection and logging in internal/health/state.go (FR-007: log transitions)
- [ ] T020 [US1] Implement Monitor struct and health check loop in internal/monitor/monitor.go (FR-001, FR-003: multiple mounts, 30s interval)
- [ ] T021 [US1] Wire up monitor in main.go entry point in cmd/mount-monitor/main.go

**Checkpoint**: User Story 1 complete - monitor detects and logs mount health changes

---

## Phase 4: User Story 2 - Pod Restart via Health Check Failure (Priority: P2)

**Goal**: Expose liveness probe endpoint that returns HTTP 503 when mounts are unhealthy past debounce threshold

**Independent Test**: Simulate mount failure, verify /healthz/live returns 503 after debounce threshold crossed

**Depends on**: US1 (health state must be tracked)

### Tests for User Story 2

- [ ] T022 [P] [US2] Write unit tests for liveness endpoint in tests/unit/server_test.go

### Implementation for User Story 2

- [ ] T023 [US2] Implement ProbeResponse and MountStatus structs in internal/server/server.go
- [ ] T024 [US2] Implement HTTP server with /healthz/live endpoint in internal/server/server.go (FR-005, FR-015: HTTP 200/503)
- [ ] T025 [US2] Implement liveness probe logic (HEALTHY/DEGRADED=200, UNHEALTHY=503) in internal/server/server.go
- [ ] T026 [US2] Add probe response logging in internal/server/server.go (FR-008: log probe queries)
- [ ] T027 [US2] Wire up HTTP server in main.go in cmd/mount-monitor/main.go

**Checkpoint**: User Story 2 complete - liveness probe triggers pod restart on confirmed unhealthy state

---

## Phase 5: User Story 3 - Service Startup Gating (Priority: P3)

**Goal**: Expose readiness probe endpoint that returns HTTP 503 when any mount is not fully healthy

**Independent Test**: Start with unhealthy mount, verify /healthz/ready returns 503; restore mount, verify 200

**Depends on**: US1 (health state), US2 (HTTP server infrastructure)

### Tests for User Story 3

- [ ] T028 [P] [US3] Write unit tests for readiness endpoint in tests/unit/server_test.go (extend existing)

### Implementation for User Story 3

- [ ] T029 [US3] Implement /healthz/ready endpoint in internal/server/server.go (FR-006, FR-015: HTTP 200/503)
- [ ] T030 [US3] Implement readiness probe logic (only HEALTHY=200, any other state=503) in internal/server/server.go

**Checkpoint**: User Story 3 complete - readiness probe gates service startup until all mounts healthy

---

## Phase 6: User Story 4 - Graceful Shutdown (Priority: P4)

**Goal**: Handle SIGTERM/SIGINT signals, complete in-flight operations, exit cleanly within 30s

**Independent Test**: Send SIGTERM to running monitor, verify clean exit with code 0 within 30s

**Depends on**: US1 (monitor loop), US2 (HTTP server)

### Tests for User Story 4

- [x] T031 [P] [US4] Write integration test for graceful shutdown in tests/unit/shutdown_test.go

### Implementation for User Story 4

- [ ] T032 [US4] Implement signal handler for SIGTERM/SIGINT in cmd/mount-monitor/main.go (FR-009)
- [ ] T033 [US4] Implement graceful HTTP server shutdown in internal/server/server.go
- [ ] T034 [US4] Implement graceful monitor loop shutdown in internal/monitor/monitor.go
- [ ] T035 [US4] Implement shutdown timeout (30s) and exit code handling in cmd/mount-monitor/main.go (FR-010, FR-014)
- [ ] T036 [US4] Add shutdown logging in cmd/mount-monitor/main.go

**Checkpoint**: User Story 4 complete - service shuts down gracefully on termination signals

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final integration, documentation, and validation

- [x] T037 [P] Write end-to-end integration test in tests/unit/monitor_test.go
- [ ] T038 Validate quickstart.md scenarios work correctly
- [ ] T039 [P] Update README.md with build and usage instructions
- [ ] T040 Run full test suite and verify CI passes
- [ ] T041 Build and verify container image size (<20MB target)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - US1 can start immediately after Foundational
  - US2 depends on US1 (needs health state)
  - US3 depends on US2 (extends HTTP server)
  - US4 depends on US1 + US2 (shuts down both)
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

```
Phase 1: Setup
    │
    ▼
Phase 2: Foundational
    │
    ▼
Phase 3: US1 (Health Monitoring) ─────────────┐
    │                                         │
    ▼                                         │
Phase 4: US2 (Liveness Probe) ────────────────┤
    │                                         │
    ▼                                         │
Phase 5: US3 (Readiness Probe)                │
    │                                         │
    └────────────────┬────────────────────────┘
                     │
                     ▼
Phase 6: US4 (Graceful Shutdown)
    │
    ▼
Phase 7: Polish
```

### Within Each User Story

- Tests written first (marked [P] within story)
- Models/structs before logic
- Core implementation before integration
- Wire-up to main.go last

### Parallel Opportunities

- All Setup tasks T003-T006 can run in parallel
- Foundational tasks T009, T010, T012 can run in parallel
- Tests within each story (T013/T014, T022, T028, T031) can run in parallel
- Polish tasks T037, T039 can run in parallel

---

## Parallel Example: Setup Phase

```bash
# Launch all parallelizable setup tasks together:
Task: "Create Dockerfile for multi-stage build in build/Dockerfile"
Task: "Create debug Dockerfile with Alpine base in build/Dockerfile.debug"
Task: "Create GitHub Actions CI workflow in .github/workflows/ci.yml"
Task: "Create .gitignore for Go project in repository root"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test health monitoring independently
5. Monitor can detect and log mount health changes (MVP value delivered)

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Health monitoring works → Can observe mount state
3. Add User Story 2 → Liveness probe works → Pod restarts on failure
4. Add User Story 3 → Readiness probe works → Startup gating enabled
5. Add User Story 4 → Graceful shutdown works → Production ready
6. Each story adds value without breaking previous stories

### Sequential Execution (Single Developer)

Due to story dependencies, execute in order: US1 → US2 → US3 → US4

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- US2 depends on US1; US3 depends on US2; US4 depends on US1+US2
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- All file paths are relative to repository root
