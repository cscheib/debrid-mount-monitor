// Package watchdog provides pod-level restart capabilities when mount health checks fail.
// It monitors mount state transitions and triggers Kubernetes pod deletion to ensure
// all containers in a pod restart together with fresh mount connections.
package watchdog

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"
)

// WatchdogStatus represents the current state of the watchdog state machine.
type WatchdogStatus int

const (
	// WatchdogDisabled indicates watchdog mode is off (default).
	WatchdogDisabled WatchdogStatus = iota
	// WatchdogArmed indicates the watchdog is monitoring for unhealthy mounts.
	WatchdogArmed
	// WatchdogPendingRestart indicates the watchdog is waiting for restart delay to elapse.
	WatchdogPendingRestart
	// WatchdogTriggered indicates pod deletion is in progress.
	WatchdogTriggered
)

// String returns a human-readable representation of the watchdog status.
func (s WatchdogStatus) String() string {
	switch s {
	case WatchdogDisabled:
		return "disabled"
	case WatchdogArmed:
		return "armed"
	case WatchdogPendingRestart:
		return "pending_restart"
	case WatchdogTriggered:
		return "triggered"
	default:
		return "unknown"
	}
}

// WatchdogState represents the current state of the watchdog.
type WatchdogState struct {
	// State is the current watchdog state machine state.
	State WatchdogStatus
	// UnhealthySince records when the mount first became unhealthy (nil if healthy).
	UnhealthySince *time.Time
	// PendingMount is the mount path that triggered the pending restart.
	PendingMount string
	// RetryCount is the current API retry attempt count.
	RetryCount int
	// LastError is the last error encountered (for logging).
	LastError error
}

// RestartEvent represents a watchdog-triggered restart for logging and Kubernetes events.
type RestartEvent struct {
	// Timestamp is when the restart was triggered.
	Timestamp time.Time
	// PodName is the name of the pod being restarted.
	PodName string
	// Namespace is the Kubernetes namespace.
	Namespace string
	// MountPath is the mount that triggered the restart.
	MountPath string
	// Reason is a human-readable reason for the restart.
	Reason string
	// FailureCount is the number of consecutive failures before trigger.
	FailureCount int
	// UnhealthyDuration is how long the mount was unhealthy.
	UnhealthyDuration time.Duration
}

// Config holds the watchdog configuration.
type Config struct {
	Enabled             bool
	RestartDelay        time.Duration
	MaxRetries          int
	RetryBackoffInitial time.Duration
	RetryBackoffMax     time.Duration
}

// Watchdog monitors mount health and triggers pod restarts when mounts become unhealthy.
type Watchdog struct {
	mu     sync.Mutex
	state  WatchdogState
	logger *slog.Logger

	// Configuration
	config Config

	// Dependencies
	k8sClient *K8sClient
	podName   string
	namespace string

	// Restart delay timer (can be cancelled)
	cancelRestart chan struct{}
	restartTimer  *time.Timer

	// Exit function (for testing)
	exitFunc func(code int)
}

// NewWatchdog creates a new Watchdog instance.
// If not running in Kubernetes or RBAC permissions are missing, the watchdog will be disabled.
func NewWatchdog(cfg Config, podName, namespace string, logger *slog.Logger) *Watchdog {
	w := &Watchdog{
		config:    cfg,
		podName:   podName,
		namespace: namespace,
		logger:    logger,
		exitFunc:  os.Exit,
		state: WatchdogState{
			State: WatchdogDisabled,
		},
	}

	return w
}

// Start initializes the watchdog, setting up the K8s client and validating permissions.
// Returns nil if watchdog is disabled or unable to start (non-fatal).
func (w *Watchdog) Start(ctx context.Context) error {
	if !w.config.Enabled {
		w.logger.Info("watchdog disabled by configuration")
		return nil
	}

	// Check if running in Kubernetes
	if !IsInCluster() {
		w.logger.Info("watchdog disabled",
			"reason", "not_in_cluster",
			"detail", "not running in kubernetes")
		return nil
	}

	// Create K8s client
	k8sClient, err := NewK8sClient(w.logger)
	if err != nil {
		w.logger.Warn("watchdog disabled",
			"reason", "k8s_client_error",
			"error", err)
		return nil
	}
	w.k8sClient = k8sClient

	// Override namespace from K8s if not set via env
	if w.namespace == "" {
		w.namespace = k8sClient.Namespace()
	}

	// Validate RBAC permissions
	canDelete, err := k8sClient.CanDeletePods(ctx)
	if err != nil {
		w.logger.Warn("watchdog disabled",
			"reason", "rbac_check_failed",
			"error", err)
		return nil
	}
	if !canDelete {
		w.logger.Warn("watchdog disabled",
			"reason", "rbac_missing",
			"detail", "missing permission to delete pods",
			"namespace", w.namespace)
		return nil
	}

	// All checks passed - arm the watchdog
	w.mu.Lock()
	w.state.State = WatchdogArmed
	w.mu.Unlock()

	w.logger.Info("watchdog armed",
		"pod", w.podName,
		"namespace", w.namespace,
		"restart_delay", w.config.RestartDelay)

	return nil
}

// OnMountUnhealthy is called when a mount transitions to unhealthy state.
// It triggers the restart sequence if the watchdog is armed.
func (w *Watchdog) OnMountUnhealthy(mountPath string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.state.State != WatchdogArmed {
		return
	}

	now := time.Now()
	w.state.UnhealthySince = &now
	w.state.PendingMount = mountPath
	w.state.State = WatchdogPendingRestart

	w.logger.Warn("watchdog restart pending",
		"mount_path", mountPath,
		"delay", w.config.RestartDelay)

	// Start the restart delay timer
	w.cancelRestart = make(chan struct{})

	if w.config.RestartDelay == 0 {
		// Immediate restart
		go w.triggerRestart()
	} else {
		// Delayed restart
		w.restartTimer = time.AfterFunc(w.config.RestartDelay, func() {
			w.triggerRestart()
		})
	}
}

// OnMountHealthy is called when a mount transitions to healthy state.
// It cancels any pending restart if the mount that triggered it recovers.
func (w *Watchdog) OnMountHealthy(mountPath string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.state.State != WatchdogPendingRestart {
		return
	}

	// Only cancel if this is the mount that triggered the pending restart
	if w.state.PendingMount != mountPath {
		return
	}

	w.logger.Info("watchdog restart cancelled",
		"mount_path", mountPath,
		"reason", "mount_recovered")

	// Cancel the pending restart
	if w.restartTimer != nil {
		w.restartTimer.Stop()
		w.restartTimer = nil
	}
	if w.cancelRestart != nil {
		close(w.cancelRestart)
		w.cancelRestart = nil
	}

	// Reset state
	w.state.State = WatchdogArmed
	w.state.UnhealthySince = nil
	w.state.PendingMount = ""
}

// triggerRestart initiates the pod deletion process.
func (w *Watchdog) triggerRestart() {
	w.mu.Lock()

	// Check if restart was cancelled
	if w.state.State != WatchdogPendingRestart {
		w.mu.Unlock()
		return
	}

	w.state.State = WatchdogTriggered
	mountPath := w.state.PendingMount
	unhealthySince := w.state.UnhealthySince
	w.mu.Unlock()

	var unhealthyDuration time.Duration
	if unhealthySince != nil {
		unhealthyDuration = time.Since(*unhealthySince)
	}

	ctx := context.Background()

	// Check if pod is already terminating
	isTerminating, err := w.k8sClient.IsPodTerminating(ctx, w.podName)
	if err != nil {
		w.logger.Warn("failed to check pod termination status",
			"error", err)
		// Continue with deletion attempt anyway
	} else if isTerminating {
		w.logger.Info("pod already terminating, skipping deletion",
			"pod", w.podName)
		return
	}

	// Create restart event
	event := &RestartEvent{
		Timestamp:         time.Now(),
		PodName:           w.podName,
		Namespace:         w.namespace,
		MountPath:         mountPath,
		Reason:            "Mount " + mountPath + " unhealthy, triggering pod restart",
		UnhealthyDuration: unhealthyDuration,
	}

	w.logger.Warn("watchdog restart triggered",
		"mount_path", mountPath,
		"unhealthy_duration", unhealthyDuration,
		"pod", w.podName)

	// Create Kubernetes event (best effort)
	if err := w.k8sClient.CreateEvent(ctx, event); err != nil {
		w.logger.Warn("failed to create kubernetes event",
			"error", err)
	}

	// Attempt pod deletion with retries
	w.deletePodWithRetry(ctx, event)
}

// deletePodWithRetry attempts to delete the pod with exponential backoff.
func (w *Watchdog) deletePodWithRetry(ctx context.Context, event *RestartEvent) {
	backoff := w.config.RetryBackoffInitial

	for attempt := 1; attempt <= w.config.MaxRetries; attempt++ {
		err := w.k8sClient.DeletePod(ctx, w.podName)
		if err == nil {
			// Success - pod deletion initiated
			w.logger.Info("pod deletion successful",
				"pod", w.podName,
				"attempt", attempt)
			return
		}

		// Check if error is permanent (should not retry)
		if _, isPermanent := err.(*PermanentError); isPermanent {
			w.logger.Error("pod deletion failed with permanent error",
				"error", err,
				"pod", w.podName)
			break
		}

		// Log retry attempt
		w.logger.Warn("pod deletion failed, retrying",
			"error", err,
			"attempt", attempt,
			"max_retries", w.config.MaxRetries,
			"next_backoff", backoff)

		w.mu.Lock()
		w.state.RetryCount = attempt
		w.state.LastError = err
		w.mu.Unlock()

		// Wait before retry
		time.Sleep(backoff)

		// Exponential backoff
		backoff *= 2
		if backoff > w.config.RetryBackoffMax {
			backoff = w.config.RetryBackoffMax
		}
	}

	// All retries exhausted - fall back to process exit
	w.logger.Error("pod deletion failed after all retries, exiting for container restart",
		"retries", w.config.MaxRetries,
		"pod", w.podName)

	w.exitFunc(1)
}

// IsEnabled returns true if the watchdog is enabled and armed.
func (w *Watchdog) IsEnabled() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.state.State != WatchdogDisabled
}

// State returns the current watchdog state.
func (w *Watchdog) State() WatchdogState {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.state
}

// SetExitFunc sets the function to call when the watchdog needs to exit.
// Used for testing to avoid actually exiting the process.
func (w *Watchdog) SetExitFunc(f func(int)) {
	w.exitFunc = f
}
