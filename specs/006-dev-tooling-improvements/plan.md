# Implementation Plan: Development Tooling Improvements

**Branch**: `006-dev-tooling-improvements` | **Date**: 2025-12-16 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/006-dev-tooling-improvements/spec.md`

## Summary

Enhance development workflow with automated end-to-end testing in KIND, comprehensive watchdog unit tests (80%+ coverage), operator troubleshooting documentation, and namespace customization support. This feature builds on the existing KIND infrastructure (spec-004) and watchdog implementation (spec-005) to improve developer experience and test confidence.

## Technical Context

**Language/Version**: Go 1.21+ (existing project standard)
**Primary Dependencies**: Standard library only (no external dependencies per constitution)
**Storage**: N/A (test infrastructure, no persistent storage)
**Testing**: Go testing package (`go test`), shell scripts for e2e automation
**Target Platform**: Linux (KIND/Docker), macOS (development)
**Project Type**: Single project with CLI tooling
**Performance Goals**: `make kind-test` completes in <5 minutes
**Constraints**: Tests must be idempotent and cleanup after themselves
**Scale/Scope**: Developer tooling - single developer workflow optimization

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Minimal Dependencies | ✅ PASS | Shell scripts + Go standard library only |
| II. Single Static Binary | ✅ N/A | Tooling/tests, not binary changes |
| III. Cross-Platform Compilation | ✅ N/A | Tooling only, binary unchanged |
| IV. Signal Handling | ✅ N/A | No signal handling changes |
| V. Container Sidecar Design | ✅ PASS | Tests validate sidecar behavior |
| VI. Fail-Safe Orchestration | ✅ PASS | Tests verify watchdog fail-safe behavior |

**Gate Result**: PASS - No violations, tooling-only feature.

## Project Structure

### Documentation (this feature)

```text
specs/006-dev-tooling-improvements/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output (minimal - no data model)
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (empty - no APIs)
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
# Existing structure - no new directories
Makefile                        # Add kind-test target
deploy/kind/                    # Add test-config variant
├── configmap.yaml
├── deployment.yaml
├── namespace.yaml
├── rbac.yaml
├── README.md
├── service.yaml
└── test-configmap.yaml        # NEW: Test-specific config (short delays)

scripts/                        # NEW: Test automation scripts
└── kind-e2e-test.sh           # NEW: E2E test orchestration

tests/unit/
└── watchdog_test.go           # MODIFY: Add comprehensive tests

docs/                           # NEW: Documentation directory
└── troubleshooting.md         # NEW: Operator troubleshooting guide
```

**Structure Decision**: Extend existing Makefile with `kind-test` target. Add minimal new directories (`scripts/`, `docs/`) following project conventions. No changes to Go package structure.

## Complexity Tracking

> No violations to justify - tooling feature with no architectural complexity.

| Item | Decision | Rationale |
|------|----------|-----------|
| Test script location | `scripts/kind-e2e-test.sh` | Separates from .specify scripts, follows Unix convention |
| Docs location | `docs/troubleshooting.md` | Standard location for project documentation |
| Test config | `deploy/kind/test-configmap.yaml` | Keeps test variants alongside production manifests |

## Key Design Decisions

### D1: E2E Test Orchestration

**Decision**: Shell script (`scripts/kind-e2e-test.sh`) called by Makefile target

**Rationale**:
- Shell scripts are the natural choice for orchestrating kubectl, kind, and Docker commands
- Makefile provides developer-friendly interface (`make kind-test`)
- Script handles idempotency, cleanup, and timeout logic
- Aligns with existing KIND targets pattern in Makefile

**Alternatives Considered**:
- Go test framework: Would require K8s client-go dependency (violates constitution)
- Python script: Adds runtime dependency (violates constitution)
- Makefile-only: Too complex for timeout/retry logic

### D2: Test Configuration Approach

**Decision**: Separate `test-configmap.yaml` with short delays (5s restart delay)

**Rationale**:
- Keeps production config unchanged
- Test-specific tuning (shorter delays for faster feedback)
- Simple swap via kubectl apply
- No impact on production deployments

### D3: Namespace Customization

**Decision**: `KIND_NAMESPACE` environment variable (already partially supported)

**Rationale**:
- Consistent with existing `KIND_CLUSTER_NAME` pattern
- No code changes to Go application
- Only affects deployment scripts and Makefile
- Enables parallel test runs with different namespaces

### D4: Mock K8sClient for Unit Tests

**Decision**: Interface-based mocking in test file

**Rationale**:
- Go interface makes dependency injection natural
- Mock lives in test file, not production code
- Enables testing all error paths without actual K8s cluster
- Follows existing test patterns in project

## Implementation Phases

### Phase 1: KIND Namespace Customization (US4)
- Update Makefile to use `KIND_NAMESPACE` variable consistently
- Update deployment commands to create namespace if not exists
- Test parallel deployments with different namespaces

### Phase 2: Test Infrastructure (US1 Setup)
- Create `scripts/kind-e2e-test.sh` with basic structure
- Create `deploy/kind/test-configmap.yaml` with short delays
- Add `make kind-test` target to Makefile

### Phase 3: E2E Test Implementation (US1 Core)
- Implement cluster setup/teardown in script
- Implement mount failure simulation
- Implement pod restart verification
- Implement WatchdogRestart event verification
- Add timeout and retry logic

### Phase 4: Unit Test Expansion (US2)
- Add mock K8sClient implementation
- Add state transition tests (Armed → PendingRestart → Triggered)
- Add DeletePod retry/backoff tests
- Add restart cancellation tests
- Add RBAC validation failure tests
- Verify 80%+ coverage

### Phase 5: Documentation (US3)
- Create `docs/troubleshooting.md`
- Document RBAC issues and resolution
- Document POD_NAME/POD_NAMESPACE configuration
- Document mount detection issues
- Document pod restart failures
- Include diagnostic commands for each issue

### Phase 6: Integration & Validation
- Run full test suite
- Verify all acceptance scenarios
- Update README with new targets
- Final cleanup and polish
