# Tasks: Pod Restart Watchdog

**Input**: Design documents from `/specs/005-pod-restart-watchdog/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/ ✓

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md structure:
- **Go source**: `cmd/`, `internal/` at repository root
- **Tests**: `tests/unit/`
- **Kubernetes manifests**: `deploy/kind/`

---

## Phase 1: Setup

**Purpose**: Create new package structure for watchdog feature

- [x] T001 Create watchdog package directory at internal/watchdog/
- [x] T002 [P] Create placeholder watchdog.go in internal/watchdog/watchdog.go
- [x] T003 [P] Create placeholder k8s_client.go in internal/watchdog/k8s_client.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Add WatchdogConfig fields to internal/config/config.go (WatchdogEnabled, RestartDelay, MaxRetries, RetryBackoffInitial, RetryBackoffMax)
- [x] T005 Add watchdog config validation rules to Validate() in internal/config/config.go
- [x] T006 Extend JSON parsing for watchdog section in internal/config/file.go
- [x] T007 Add environment variable support for WATCHDOG_ENABLED and WATCHDOG_RESTART_DELAY in internal/config/config.go
- [x] T008 [P] Add WatchdogStatus enum (Disabled, Armed, PendingRestart, Triggered) to internal/watchdog/watchdog.go
- [x] T009 [P] Add WatchdogState struct with fields (State, UnhealthySince, PendingMount, RetryCount, LastError) to internal/watchdog/watchdog.go
- [x] T010 [P] Add RestartEvent struct with fields (Timestamp, PodName, Namespace, MountPath, Reason, FailureCount, UnhealthyDuration) to internal/watchdog/watchdog.go

**Checkpoint**: Foundation ready - config parsing works, types defined, user story implementation can begin

---

## Phase 3: User Story 1 - Mount Failure Triggers Pod Restart (Priority: P1) 🎯 MVP

**Goal**: When mount becomes UNHEALTHY, watchdog deletes pod via Kubernetes API, causing all containers to restart together

**Independent Test**: Deploy to KIND cluster, delete canary file, verify entire pod restarts (both containers get new start times)

### Implementation for User Story 1

- [x] T011 [P] [US1] Implement IsInCluster() function to detect Kubernetes environment in internal/watchdog/k8s_client.go
- [x] T012 [P] [US1] Implement LoadInClusterConfig() to read service account token, CA cert, and namespace in internal/watchdog/k8s_client.go
- [x] T013 [US1] Implement K8sClient struct with New() constructor using in-cluster auth in internal/watchdog/k8s_client.go
- [x] T014 [US1] Implement CreateHTTPClient() with TLS configuration for K8s API in internal/watchdog/k8s_client.go
- [x] T015 [US1] Implement DeletePod(ctx, name) method with proper status code handling in internal/watchdog/k8s_client.go
- [x] T016 [US1] Implement IsPodTerminating(ctx, name) method to check deletionTimestamp in internal/watchdog/k8s_client.go
- [x] T017 [US1] Implement Watchdog struct with New() constructor accepting config, logger, and K8sClient in internal/watchdog/watchdog.go
- [x] T018 [US1] Implement OnMountStateChange(mount, transition) method to trigger state machine in internal/watchdog/watchdog.go
- [x] T019 [US1] Implement state transition logic (Armed → PendingRestart → Triggered) in internal/watchdog/watchdog.go
- [x] T020 [US1] Implement triggerRestart() method that checks IsPodTerminating() first, then calls DeletePod and handles errors in internal/watchdog/watchdog.go
- [x] T021 [US1] Implement restart delay timer with cancellation support in internal/watchdog/watchdog.go
- [x] T022 [US1] Add watchdog integration to Monitor in internal/monitor/monitor.go (call OnMountStateChange on transitions)
- [x] T023 [US1] Initialize watchdog in main.go with config and pass to Monitor in cmd/mount-monitor/main.go
- [x] T024 [US1] Add POD_NAME and POD_NAMESPACE environment variable reading in cmd/mount-monitor/main.go
- [x] T025 [US1] Create RBAC manifest (ServiceAccount, Role, RoleBinding) at deploy/kind/rbac.yaml
- [x] T026 [US1] Update deployment.yaml to reference ServiceAccount and add POD_NAME/POD_NAMESPACE env vars in deploy/kind/deployment.yaml

**Checkpoint**: User Story 1 complete - mount failure triggers pod restart via K8s API

---

## Phase 4: User Story 2 - Configurable Watchdog Behavior (Priority: P2)

**Goal**: Operators can enable/disable watchdog and customize restart delay via config file or environment variables

**Independent Test**: Deploy with watchdog disabled, verify pod does NOT restart on mount failure (only readiness probe fails)

### Implementation for User Story 2

- [x] T027 [US2] Implement CanDeletePods(ctx) RBAC validation method in internal/watchdog/k8s_client.go
- [x] T028 [US2] Add RBAC validation at startup - log error and disable watchdog if permissions missing in internal/watchdog/watchdog.go
- [x] T029 [US2] Implement graceful degradation when running outside Kubernetes (IsInCluster() = false) in internal/watchdog/watchdog.go
- [x] T030 [US2] Implement handleRecovery() method to cancel pending restart when mount recovers in internal/watchdog/watchdog.go
- [x] T031 [US2] Update configmap.yaml with watchdog configuration section in deploy/kind/configmap.yaml
- [x] T032 [US2] Add unit test for watchdog disabled behavior in tests/unit/watchdog_test.go
- [x] T033 [US2] Add unit test for restart delay configuration in tests/unit/watchdog_test.go
- [x] T034 [US2] Add unit test for restart cancellation on recovery in tests/unit/watchdog_test.go

**Checkpoint**: User Story 2 complete - watchdog fully configurable, graceful degradation works

---

## Phase 5: User Story 3 - Restart Event Visibility (Priority: P3)

**Goal**: Operators can see watchdog restarts in logs and kubectl get events

**Independent Test**: Trigger watchdog restart, verify structured log with mount_path/reason/timestamp and Kubernetes event with WatchdogRestart reason

### Implementation for User Story 3

- [x] T035 [P] [US3] Implement CreateEvent(ctx, event) method to create Kubernetes Event resources in internal/watchdog/k8s_client.go
- [x] T036 [US3] Add structured logging for all watchdog state transitions (armed, pending, triggered, cancelled, failed) in internal/watchdog/watchdog.go
- [x] T037 [US3] Create Kubernetes event before pod deletion with reason "WatchdogRestart" in internal/watchdog/watchdog.go
- [x] T038 [US3] Implement exponential backoff retry logic for DeletePod with configurable max retries in internal/watchdog/watchdog.go
- [x] T039 [US3] Implement fallback to os.Exit(1) after retries exhausted in internal/watchdog/watchdog.go
- [x] T040 [US3] Add logging for retry attempts with attempt count and error details in internal/watchdog/watchdog.go
- [x] T041 [US3] Add unit test for Kubernetes event creation in tests/unit/watchdog_test.go
- [x] T042 [US3] Add unit test for exponential backoff retry logic in tests/unit/watchdog_test.go

**Checkpoint**: User Story 3 complete - full observability for watchdog restarts

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, validation, and cleanup

- [x] T043 [P] Update quickstart.md with actual test commands after deployment in specs/005-pod-restart-watchdog/quickstart.md
- [x] T044 [P] Add watchdog section to project README.md
- [x] T045 Run KIND deployment and validate end-to-end watchdog behavior per quickstart.md
- [x] T046 Verify all acceptance scenarios from spec.md pass
- [x] T047 Code cleanup and go fmt/vet/lint validation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - US1 (P1): Core restart functionality
  - US2 (P2): Configuration and graceful degradation
  - US3 (P3): Observability and retry logic
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Builds on US1 K8sClient, adds RBAC validation and config handling
- **User Story 3 (P3)**: Builds on US1/US2, adds events and retry logic

### Within Each User Story

- K8sClient methods before Watchdog methods
- Watchdog core before Monitor integration
- Core implementation before unit tests
- Code before Kubernetes manifests

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- T008, T009, T010 (type definitions) can run in parallel
- T011, T012 (K8s detection functions) can run in parallel
- T035 (CreateEvent) can run in parallel with other US3 tasks

---

## Parallel Example: User Story 1 K8s Client

```bash
# Launch K8s detection functions together:
Task: "Implement IsInCluster() function in internal/watchdog/k8s_client.go"
Task: "Implement LoadInClusterConfig() in internal/watchdog/k8s_client.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (create package structure)
2. Complete Phase 2: Foundational (config parsing, type definitions)
3. Complete Phase 3: User Story 1 (core pod deletion)
4. **STOP and VALIDATE**: Deploy to KIND, test pod restart
5. Deploy/demo if ready - MVP complete!

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test in KIND → MVP with core restart
3. Add User Story 2 → Test config options → Full configuration
4. Add User Story 3 → Test events/logs → Full observability
5. Each story adds value without breaking previous stories

### Task Count by Phase

| Phase | Tasks | Parallel Tasks |
|-------|-------|----------------|
| Setup | 3 | 2 |
| Foundational | 7 | 3 |
| User Story 1 (P1) | 16 | 2 |
| User Story 2 (P2) | 8 | 0 |
| User Story 3 (P3) | 8 | 1 |
| Polish | 5 | 2 |
| **Total** | **47** | **10** |

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
- Tests included per user story (unit tests for new watchdog package)
