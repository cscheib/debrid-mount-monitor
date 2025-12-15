// Package health provides mount health checking and state management.
package health

import (
	"path/filepath"
	"sync"
	"time"
)

// HealthStatus represents the health state of a mount.
type HealthStatus int

const (
	// StatusUnknown is the initial state before any check is performed.
	StatusUnknown HealthStatus = iota
	// StatusHealthy indicates the mount is accessible.
	StatusHealthy
	// StatusDegraded indicates the mount has failed but is within debounce threshold.
	StatusDegraded
	// StatusUnhealthy indicates the mount has failed past debounce threshold.
	StatusUnhealthy
)

// String returns the string representation of the health status.
func (s HealthStatus) String() string {
	switch s {
	case StatusUnknown:
		return "unknown"
	case StatusHealthy:
		return "healthy"
	case StatusDegraded:
		return "degraded"
	case StatusUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// Mount represents a single mount point being monitored.
type Mount struct {
	Path         string       // Absolute path to mount point
	CanaryPath   string       // Full path to canary file
	Status       HealthStatus // Current health status
	LastCheck    time.Time    // Timestamp of last health check
	LastError    error        // Last error encountered (nil if healthy)
	FailureCount int          // Consecutive failure count for debounce
	mu           sync.RWMutex // Protects all fields
}

// NewMount creates a new Mount instance.
func NewMount(path, canaryFile string) *Mount {
	canaryPath := path
	if canaryFile != "" {
		canaryPath = filepath.Join(path, canaryFile)
	}
	return &Mount{
		Path:       path,
		CanaryPath: canaryPath,
		Status:     StatusUnknown,
	}
}

// GetStatus returns the current health status thread-safely.
func (m *Mount) GetStatus() HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Status
}

// GetFailureCount returns the current failure count thread-safely.
func (m *Mount) GetFailureCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.FailureCount
}

// GetLastCheck returns the last check time thread-safely.
func (m *Mount) GetLastCheck() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.LastCheck
}

// GetLastError returns the last error thread-safely.
func (m *Mount) GetLastError() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.LastError
}

// CheckResult represents the outcome of a single health check.
type CheckResult struct {
	Mount     *Mount        // Reference to the mount checked
	Timestamp time.Time     // When the check was performed
	Success   bool          // Whether the canary file was readable
	Duration  time.Duration // How long the check took
	Error     error         // Error if check failed (nil on success)
}

// StateTransition records a change in health status.
type StateTransition struct {
	Mount         *Mount       // Reference to the mount
	Timestamp     time.Time    // When the transition occurred
	PreviousState HealthStatus // State before transition
	NewState      HealthStatus // State after transition
	Trigger       string       // What caused the transition
}

// UpdateState updates the mount's state based on a check result.
// Returns a StateTransition if the state changed, nil otherwise.
func (m *Mount) UpdateState(result *CheckResult, debounceThreshold int) *StateTransition {
	m.mu.Lock()
	defer m.mu.Unlock()

	previousState := m.Status
	m.LastCheck = result.Timestamp

	if result.Success {
		// Check passed - reset to healthy
		m.FailureCount = 0
		m.LastError = nil
		m.Status = StatusHealthy
	} else {
		// Check failed
		m.FailureCount++
		m.LastError = result.Error

		if m.FailureCount >= debounceThreshold {
			m.Status = StatusUnhealthy
		} else {
			m.Status = StatusDegraded
		}
	}

	// Return transition if state changed
	if m.Status != previousState {
		trigger := "check_passed"
		if !result.Success {
			trigger = "check_failed"
		}
		if previousState == StatusUnhealthy && m.Status == StatusHealthy {
			trigger = "recovered"
		}
		return &StateTransition{
			Mount:         m,
			Timestamp:     result.Timestamp,
			PreviousState: previousState,
			NewState:      m.Status,
			Trigger:       trigger,
		}
	}

	return nil
}

// Snapshot returns a thread-safe copy of mount state for reporting.
type MountSnapshot struct {
	Path         string
	Status       HealthStatus
	LastCheck    time.Time
	FailureCount int
	LastError    string
}

// Snapshot returns a point-in-time copy of the mount's state.
func (m *Mount) Snapshot() MountSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	errStr := ""
	if m.LastError != nil {
		errStr = m.LastError.Error()
	}

	return MountSnapshot{
		Path:         m.Path,
		Status:       m.Status,
		LastCheck:    m.LastCheck,
		FailureCount: m.FailureCount,
		LastError:    errStr,
	}
}
