# Feature Specification: Testing Quality Improvements

**Feature Branch**: `011-testing-improvements`
**Created**: 2025-12-18
**Status**: Draft
**Input**: User description: "Improve testing quality and effectiveness by adding goroutine leak detection, race condition testing, coverage reporting, and expanding unit and E2E test coverage"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Developer Detects Goroutine Leaks (Priority: P1)

A developer working on the mount-monitor codebase needs confidence that their code changes don't introduce goroutine leaks, particularly important given the known risk of hung NFS mounts causing goroutine accumulation.

**Why this priority**: Goroutine leaks are a documented concern in this codebase. The health checker uses goroutine-per-check patterns that can leak when NFS mounts hang. Detecting these issues early prevents production memory exhaustion.

**Independent Test**: Can be fully tested by running the test suite with goleak integration and verifying zero leaked goroutines are reported.

**Acceptance Scenarios**:

1. **Given** a developer runs the test suite, **When** any test leaves goroutines running after completion, **Then** the test fails with a clear message identifying the leaked goroutine(s)
2. **Given** a developer writes a test for code that spawns goroutines, **When** the goroutines are properly cleaned up, **Then** the test passes without leak warnings
3. **Given** the existing test suite, **When** all tests are run with leak detection enabled, **Then** no existing tests fail due to pre-existing leaks (baseline is clean)

---

### User Story 2 - Developer Identifies Race Conditions (Priority: P1)

A developer needs to verify that concurrent code in the monitor, health state, and watchdog components is free from data races that could cause unpredictable behavior.

**Why this priority**: The codebase has multiple concurrent components (monitor loop, health state with RWMutex, watchdog timers). Race conditions can cause silent data corruption that only manifests in production under load.

**Independent Test**: Can be tested by running `go test -race` and verifying zero race conditions are detected.

**Acceptance Scenarios**:

1. **Given** a developer runs the test suite with race detection, **When** the tests complete, **Then** zero race conditions are reported
2. **Given** concurrent access tests for shared state, **When** multiple goroutines access the state simultaneously, **Then** no races occur and data remains consistent
3. **Given** the CI pipeline, **When** tests are run, **Then** race detection is automatically included

---

### User Story 3 - Developer Tracks Test Coverage (Priority: P2)

A developer wants visibility into which parts of the codebase are tested and which are not, enabling informed decisions about where to add tests.

**Why this priority**: Current coverage is 57.3% with critical gaps (watchdog 45%, main.go 0%). Visibility into coverage trends prevents regression and guides testing effort.

**Independent Test**: Can be tested by running coverage tools and verifying reports are generated with accurate per-package breakdowns.

**Acceptance Scenarios**:

1. **Given** a developer runs the test suite with coverage, **When** tests complete, **Then** a coverage report is generated showing per-package coverage percentages
2. **Given** the CI pipeline, **When** a pull request is submitted, **Then** coverage data is reported (without blocking on thresholds initially)
3. **Given** the Makefile, **When** a developer runs the coverage target, **Then** an HTML coverage report is generated for detailed analysis

---

### User Story 4 - Developer Has Confidence in Watchdog Behavior (Priority: P2)

A developer modifying the watchdog component needs comprehensive tests that verify the state machine, timer cancellation, and pod restart logic work correctly under all conditions.

**Why this priority**: The watchdog manages pod restarts in production - bugs here cause service outages. Current coverage is 45.2%, leaving critical paths untested.

**Independent Test**: Can be tested by running watchdog unit tests and verifying all state transitions, timer behaviors, and edge cases pass.

**Acceptance Scenarios**:

1. **Given** a mount enters unhealthy state, **When** the restart delay expires, **Then** the watchdog transitions to triggered state and initiates pod deletion
2. **Given** a pending restart, **When** the mount recovers before the timer expires, **Then** the restart is cancelled and watchdog returns to armed state
3. **Given** multiple concurrent mount state changes, **When** failure and recovery events race, **Then** the watchdog handles them correctly without deadlock or invalid states
4. **Given** API calls fail, **When** retries are exhausted, **Then** the appropriate fallback behavior occurs (process exit)

---

### User Story 5 - Developer Tests Entry Point Logic (Priority: P2)

A developer needs confidence that the application entry point (main.go) correctly initializes components, handles signals, and coordinates shutdown.

**Why this priority**: Currently 0% tested. Bugs in startup or shutdown logic cause deployment failures and data loss during restarts.

**Independent Test**: Can be tested by running main.go tests that verify logger setup, init-mode execution, and shutdown coordination.

**Acceptance Scenarios**:

1. **Given** the logger setup function, **When** it is invoked, **Then** debug/info logs route to stdout and warn/error logs route to stderr
2. **Given** init-container mode, **When** the application runs, **Then** it performs health checks and exits with appropriate code without starting the server
3. **Given** a running application, **When** SIGTERM is received, **Then** graceful shutdown occurs in the correct order (HTTP stop, monitor cancel, monitor wait)

---

### User Story 6 - Developer Runs Comprehensive E2E Tests (Priority: P3)

A developer needs E2E tests that verify the complete system behavior in a realistic Kubernetes environment, including mount failure scenarios and recovery.

**Why this priority**: Unit tests can't catch integration issues between components. E2E tests in KIND provide confidence that the full system works correctly.

**Independent Test**: Can be tested by running KIND cluster tests and verifying mount failure detection, recovery, and pod restart scenarios work end-to-end.

**Acceptance Scenarios**:

1. **Given** a healthy mount in KIND, **When** the canary file is removed (simulating failure), **Then** the mount transitions to unhealthy and the watchdog triggers a restart
2. **Given** a failed mount, **When** the canary file is restored, **Then** the mount recovers and the pending restart is cancelled
3. **Given** the pod restart is triggered, **When** the Kubernetes API is called, **Then** the pod is deleted and an event is recorded

---

### User Story 7 - Developer Uses Shared Test Utilities (Priority: P3)

A developer writing new tests needs reusable test helpers to reduce boilerplate and ensure consistent testing patterns.

**Why this priority**: Test utilities are currently duplicated across packages. Consolidation improves maintainability and makes tests easier to write.

**Independent Test**: Can be tested by importing the testutil package and using its helpers in a new test.

**Acceptance Scenarios**:

1. **Given** the testutil package, **When** a developer needs a silent logger for tests, **Then** `testutil.Logger()` provides one without configuration
2. **Given** the testutil package, **When** a developer needs a temporary config file, **Then** `testutil.TempConfig()` creates a valid configuration
3. **Given** the testutil package, **When** a developer needs to poll for a condition, **Then** `testutil.PollUntil()` handles the polling logic with timeouts

---

### Edge Cases

- What happens when goleak detects a leak in a test that uses `t.Parallel()`?
- How does race detection interact with tests that use timeouts?
- What happens when coverage tools encounter generated or vendored code?
- How does the test suite behave when KIND cluster is unavailable for E2E tests?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Test suite MUST detect goroutine leaks using goleak library
- **FR-002**: Test suite MUST run with race detection enabled via `-race` flag
- **FR-003**: Test suite MUST generate coverage reports per package
- **FR-004**: CI pipeline MUST report coverage data on pull requests
- **FR-005**: Watchdog unit tests MUST achieve 80% or higher coverage
- **FR-006**: Entry point (main.go) tests MUST achieve 60% or higher coverage
- **FR-007**: Makefile MUST provide targets for race testing and coverage reporting
- **FR-008**: Test suite MUST include concurrent access tests for health.State
- **FR-009**: E2E tests MUST verify mount failure and recovery scenarios in KIND
- **FR-010**: Shared test utilities MUST be consolidated in testutil package
- **FR-011**: All existing tests MUST pass with goleak and race detection enabled

### Key Entities

- **Test Suite**: Collection of unit, integration, and E2E tests with coverage tracking
- **Coverage Report**: Per-package test coverage percentages and detailed HTML analysis
- **Test Utilities**: Shared helpers for logging, configuration, and polling in tests
- **KIND Cluster**: Local Kubernetes environment for E2E testing

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Overall test coverage increases from 57% to 75% or higher
- **SC-002**: Watchdog package coverage increases from 45% to 80% or higher
- **SC-003**: Entry point (main.go) coverage increases from 0% to 60% or higher
- **SC-004**: Zero race conditions detected when running with race detection
- **SC-005**: Zero goroutine leaks detected when running with goleak
- **SC-006**: All CI builds include coverage reporting
- **SC-007**: E2E tests successfully verify mount failure and recovery scenarios
- **SC-008**: Test execution with race detection completes within acceptable time limits (less than 5x normal test time)

## Assumptions

- The `go.uber.org/goleak` library meets project constitution criteria (zero transitive deps, test-only, no CGO)
- Existing KIND infrastructure from feature 006 is functional and can be extended
- The `-race` flag is compatible with all existing tests without modification
- Test coverage tools are part of the standard Go toolchain (no external dependencies)
- Refactoring main.go to extract testable functions is acceptable for improving testability
