# Tasks: KIND Local Development Environment

**Input**: Design documents from `/specs/004-kind-local-dev/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅

**Tests**: Manual verification only (no automated tests for infrastructure feature)

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Project root**: `deploy/kind/` for Kubernetes manifests
- **Makefile**: Repository root `Makefile`
- **Documentation**: `deploy/kind/README.md`

---

## Phase 1: Setup (Directory Structure)

**Purpose**: Create project structure for KIND configuration files

- [x] T001 Create deploy/kind/ directory structure at repository root
- [x] T002 Copy KIND cluster config from specs/004-kind-local-dev/contracts/kind-config.yaml to deploy/kind/kind-config.yaml

---

## Phase 2: Foundational (Kubernetes Manifests)

**Purpose**: Create all Kubernetes manifest files that deployment depends on

**⚠️ CRITICAL**: Manifests must exist before Makefile targets can deploy them

- [x] T003 [P] Copy namespace manifest from specs/004-kind-local-dev/contracts/namespace.yaml to deploy/kind/namespace.yaml
- [x] T004 [P] Copy configmap manifest from specs/004-kind-local-dev/contracts/configmap.yaml to deploy/kind/configmap.yaml
- [x] T005 [P] Copy deployment manifest from specs/004-kind-local-dev/contracts/deployment.yaml to deploy/kind/deployment.yaml

**Checkpoint**: All manifests in place - Makefile targets can now reference them

---

## Phase 3: User Story 1 - Local KIND Cluster Setup (Priority: P1) 🎯 MVP

**Goal**: Developer can create and delete a local KIND cluster with single commands

**Independent Test**: Run `make kind-create`, verify cluster exists with `kubectl get nodes`, then `make kind-delete` to clean up

### Implementation for User Story 1

- [x] T006 [US1] Add KIND_CLUSTER_NAME variable to Makefile with default value 'debrid-mount-monitor'
- [x] T007 [US1] Add `kind-create` target to Makefile that creates cluster using deploy/kind/kind-config.yaml
- [x] T008 [US1] Add `kind-delete` target to Makefile that deletes the KIND cluster (use `kind delete cluster --name` which handles corrupted clusters gracefully)
- [x] T009 [US1] Add `kind-status` target to Makefile that shows cluster and node status
- [x] T010 [US1] Add Docker availability check to `kind-create` with clear error message

**Checkpoint**: Developer can create/delete KIND cluster - User Story 1 complete

---

## Phase 4: User Story 2 - Deploy Monitor to Local Cluster (Priority: P2)

**Goal**: Developer can build, load, and deploy the mount-monitor sidecar to KIND

**Independent Test**: After `make kind-create`, run `make kind-load kind-deploy`, verify pods running with `kubectl -n mount-monitor-dev get pods`

**Dependencies**: Requires User Story 1 (cluster must exist)

### Implementation for User Story 2

- [x] T011 [US2] Add `kind-load` target to Makefile that builds image and loads into KIND cluster
- [x] T012 [US2] Add `kind-deploy` target to Makefile that applies namespace, configmap, and deployment manifests
- [x] T013 [US2] Add `kind-logs` target to Makefile that tails mount-monitor container logs
- [x] T014 [US2] Add `kind-undeploy` target to Makefile that deletes the namespace and all resources

**Checkpoint**: Developer can deploy and view monitor in local cluster - User Story 2 complete

---

## Phase 5: User Story 3 - Simulate Mount Failures (Priority: P3)

**Goal**: Documentation enables developer to manually simulate mount failures and observe pod behavior

**Independent Test**: Follow documentation to remove canary file, observe readiness probe failure, restore canary, observe recovery

**Dependencies**: Requires User Story 2 (monitor must be deployed)

### Implementation for User Story 3

- [x] T015 [US3] Add "Simulating Mount Failures" section to deploy/kind/README.md with kubectl exec commands
- [x] T016 [US3] Add "Verifying Probe Behavior" section to deploy/kind/README.md showing expected pod status changes
- [x] T017 [US3] Add "Restoring Health" section to deploy/kind/README.md with recovery commands

**Checkpoint**: Documentation covers failure simulation workflow - User Story 3 complete

---

## Phase 6: User Story 4 - Quick Iteration Workflow (Priority: P4)

**Goal**: Developer can rebuild and redeploy with a single command for rapid iteration

**Independent Test**: Modify source code, run `make kind-redeploy`, verify new version running within 60 seconds

**Dependencies**: Requires User Story 2 (deployment infrastructure)

### Implementation for User Story 4

- [x] T018 [US4] Add `kind-redeploy` target to Makefile that rebuilds, reloads, and restarts deployment
- [x] T019 [US4] Add rollout status wait to `kind-redeploy` to confirm new pods are ready
- [x] T020 [US4] Add "Quick Iteration" section to deploy/kind/README.md documenting the workflow

**Checkpoint**: Developer can iterate rapidly - User Story 4 complete

---

## Phase 7: Polish & Documentation

**Purpose**: Complete documentation and final validation

- [x] T021 [P] Create deploy/kind/README.md with complete local development workflow documentation
- [x] T022 [P] Add Prerequisites section to README.md (Docker, kubectl, KIND versions)
- [x] T023 [P] Add Troubleshooting section to README.md covering common issues
- [x] T024 Add `kind-help` target to Makefile showing all KIND-related targets
- [x] T025 Add environment variables section to README.md (KIND_CLUSTER_NAME)
- [x] T026 Validate complete workflow: kind-create → kind-load → kind-deploy → kind-logs → kind-delete
- [x] T027 Update repository root README.md with link to local development documentation

---

## Phase 8: Remediation (Post-Clarification)

**Purpose**: Address spec-implementation mismatch from JSON config clarification

**Context**: FR-002 was clarified to require JSON config file mounted as ConfigMap (not environment variables). This exercises the JSON config parsing code path.

- [x] T028 [P] Update specs/004-kind-local-dev/contracts/configmap.yaml to use JSON config file format
- [x] T029 [P] Update specs/004-kind-local-dev/contracts/deployment.yaml to mount config file and use --config flag
- [x] T030 [P] Update deploy/kind/configmap.yaml to use JSON config file format
- [x] T031 [P] Update deploy/kind/deployment.yaml to mount config file and use --config flag

**Checkpoint**: Configuration now uses JSON config file per FR-002 clarification

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1 (Setup)
    │
    ▼
Phase 2 (Manifests) ─────────────────────────────────┐
    │                                                 │
    ▼                                                 │
Phase 3 (US1: Cluster) ◄──────────────────────────────┤
    │                                                 │
    ▼                                                 │
Phase 4 (US2: Deploy) ◄───────────────────────────────┤
    │                                                 │
    ├──────────────────┬──────────────────────────────┘
    │                  │
    ▼                  ▼
Phase 5 (US3)     Phase 6 (US4)
    │                  │
    └────────┬─────────┘
             │
             ▼
      Phase 7 (Polish)
```

### User Story Dependencies

| Story | Can Start After | Independent Test |
|-------|-----------------|------------------|
| US1 (Cluster) | Phase 2 complete | `make kind-create && kubectl get nodes` |
| US2 (Deploy) | US1 complete | `make kind-deploy && kubectl get pods` |
| US3 (Failure Sim) | US2 complete | Manual canary file manipulation |
| US4 (Iteration) | US2 complete | `make kind-redeploy` timing |

### Parallel Opportunities

**Phase 2** - All manifests can be created in parallel:
```
T003 (namespace.yaml) ─┐
T004 (configmap.yaml) ─┼─► All complete before Phase 3
T005 (deployment.yaml)─┘
```

**Phase 5 & 6** - Can proceed in parallel after US2:
```
US3 (Documentation) ─┐
                     ├─► Both complete before Phase 7
US4 (Redeploy)      ─┘
```

**Phase 7** - Documentation tasks can run in parallel:
```
T021 (README.md) ─────┐
T022 (Prerequisites) ─┼─► All complete before validation
T023 (Troubleshooting)┘
```

---

## Parallel Example: Phase 2 (Manifests)

```bash
# Launch all manifest tasks together:
Task: "Copy namespace manifest to deploy/kind/namespace.yaml"
Task: "Copy configmap manifest to deploy/kind/configmap.yaml"
Task: "Copy deployment manifest to deploy/kind/deployment.yaml"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T002)
2. Complete Phase 2: Manifests (T003-T005)
3. Complete Phase 3: User Story 1 (T006-T010)
4. **STOP and VALIDATE**: `make kind-create` works, cluster accessible
5. Developer can now explore Kubernetes locally

### Incremental Delivery

| Increment | User Stories | Developer Value |
|-----------|--------------|-----------------|
| MVP | US1 | Can create local K8s cluster |
| v0.2 | US1 + US2 | Can deploy and test monitor |
| v0.3 | US1-US3 | Can simulate failures |
| v1.0 | US1-US4 + Polish | Complete local dev workflow |

### Recommended Execution Order

1. T001-T002 (Setup)
2. T003-T005 in parallel (Manifests)
3. T006-T010 sequential (US1 - Makefile targets)
4. **Validate US1**: Create/delete cluster works
5. T011-T014 sequential (US2 - Makefile targets)
6. **Validate US2**: Deploy/logs work
7. T015-T017 in parallel (US3 - Documentation)
8. T018-T020 (US4 - Iteration workflow)
9. T021-T027 (Polish - most in parallel)
10. **Final validation**: Full workflow test

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- No automated tests - this is infrastructure/configuration
- Makefile targets must be added sequentially (same file)
- YAML manifests can be created in parallel (different files)
- Commit after each user story completion for incremental progress
- Stop at any checkpoint to validate independently
