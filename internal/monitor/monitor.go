// Package monitor provides the health monitoring loop for mount points.
package monitor

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/chris/debrid-mount-monitor/internal/health"
)

// WatchdogNotifier is an interface for notifying the watchdog of mount state changes.
type WatchdogNotifier interface {
	OnMountUnhealthy(mountPath string, failureCount int)
	OnMountHealthy(mountPath string)
}

// Monitor continuously checks mount health at configured intervals.
type Monitor struct {
	mounts            []*health.Mount
	checker           *health.Checker
	interval          time.Duration
	debounceThreshold int
	logger            *slog.Logger
	wg                sync.WaitGroup
	watchdog          WatchdogNotifier
}

// New creates a new Monitor instance.
func New(mounts []*health.Mount, checker *health.Checker, interval time.Duration, debounceThreshold int, logger *slog.Logger) *Monitor {
	return &Monitor{
		mounts:            mounts,
		checker:           checker,
		interval:          interval,
		debounceThreshold: debounceThreshold,
		logger:            logger,
	}
}

// SetWatchdog sets the watchdog notifier for mount state changes.
func (m *Monitor) SetWatchdog(w WatchdogNotifier) {
	m.watchdog = w
}

// Start begins the health check loop. It runs until the context is cancelled.
func (m *Monitor) Start(ctx context.Context) {
	m.wg.Add(1)
	go m.run(ctx)
}

// Wait blocks until the monitor has stopped.
func (m *Monitor) Wait() {
	m.wg.Wait()
}

func (m *Monitor) run(ctx context.Context) {
	defer m.wg.Done()

	// Perform initial check immediately
	m.checkAll(ctx)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("monitor shutting down")
			return
		case <-ticker.C:
			m.checkAll(ctx)
		}
	}
}

func (m *Monitor) checkAll(ctx context.Context) {
	for _, mount := range m.mounts {
		select {
		case <-ctx.Done():
			return
		default:
			m.checkMount(ctx, mount)
		}
	}
}

func (m *Monitor) checkMount(ctx context.Context, mount *health.Mount) {
	result := m.checker.Check(ctx, mount)
	// Use per-mount threshold if set (>0), otherwise fall back to global threshold.
	// A threshold of 0 is a sentinel meaning "use default" - this is documented in
	// config.MountConfig and allows omitting the field in JSON config files.
	threshold := mount.FailureThreshold
	if threshold == 0 {
		threshold = m.debounceThreshold
	}
	transition := mount.UpdateState(result, threshold)

	// Log check result - include name if available for easier identification
	logAttrs := []any{
		"path", mount.Path,
		"success", result.Success,
		"duration", result.Duration.String(),
		"status", mount.GetStatus().String(),
	}

	if mount.Name != "" {
		logAttrs = append(logAttrs, "name", mount.Name)
	}

	if result.Error != nil {
		logAttrs = append(logAttrs, "error", result.Error.Error())
	}

	if result.Success {
		m.logger.Debug("health check passed", logAttrs...)
	} else {
		m.logger.Warn("health check failed", logAttrs...)
	}

	// Log state transitions - include name if available
	if transition != nil {
		transitionAttrs := []any{
			"path", mount.Path,
			"previous_state", transition.PreviousState.String(),
			"new_state", transition.NewState.String(),
			"trigger", transition.Trigger,
		}
		if mount.Name != "" {
			transitionAttrs = append(transitionAttrs, "name", mount.Name)
		}
		m.logger.Info("mount state changed", transitionAttrs...)

		// Notify watchdog of state transitions
		if m.watchdog != nil {
			if transition.NewState == health.StatusUnhealthy {
				m.watchdog.OnMountUnhealthy(mount.Path, mount.GetFailureCount())
			} else if transition.NewState == health.StatusHealthy && transition.PreviousState == health.StatusUnhealthy {
				m.watchdog.OnMountHealthy(mount.Path)
			}
		}
	}
}
