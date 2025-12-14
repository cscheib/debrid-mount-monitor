// Package monitor provides the health monitoring loop for mount points.
package monitor

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/chris/debrid-mount-monitor/internal/health"
)

// Monitor continuously checks mount health at configured intervals.
type Monitor struct {
	mounts            []*health.Mount
	checker           *health.Checker
	interval          time.Duration
	debounceThreshold int
	logger            *slog.Logger
	wg                sync.WaitGroup
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
	transition := mount.UpdateState(result, m.debounceThreshold)

	// Log check result
	logAttrs := []any{
		"path", mount.Path,
		"success", result.Success,
		"duration", result.Duration.String(),
		"status", mount.GetStatus().String(),
	}

	if result.Error != nil {
		logAttrs = append(logAttrs, "error", result.Error.Error())
	}

	if result.Success {
		m.logger.Debug("health check passed", logAttrs...)
	} else {
		m.logger.Warn("health check failed", logAttrs...)
	}

	// Log state transitions
	if transition != nil {
		m.logger.Info("mount state changed",
			"path", mount.Path,
			"previous_state", transition.PreviousState.String(),
			"new_state", transition.NewState.String(),
			"trigger", transition.Trigger,
		)
	}
}
