package tagit

import (
	"context"
	"log/slog"
	"time"
)

// Scheduler repeats one reconciliation pass until its context is cancelled.
type Scheduler struct {
	Interval time.Duration
	Ticks    <-chan time.Time
	RunOnce  func() error
	Logger   *slog.Logger
}

// Run starts the scheduler loop.
func (s Scheduler) Run(ctx context.Context) {
	if s.RunOnce == nil {
		return
	}

	ticks := s.Ticks
	var ticker *time.Ticker
	if ticks == nil {
		if s.Interval <= 0 {
			s.logger().Error("invalid scheduler interval", "interval", s.Interval)
			return
		}
		ticker = time.NewTicker(s.Interval)
		defer ticker.Stop()
		ticks = ticker.C
	}

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ticks:
			if !ok {
				return
			}
			if err := s.RunOnce(); err != nil {
				s.logger().Error("error updating service tags", "error", err)
			}
		}
	}
}

func (s Scheduler) logger() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}
