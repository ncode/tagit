package tagit

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
)

func TestTagIt_ReconcileOnce(t *testing.T) {
	var gotTags []string
	tagit := New(
		&MockConsulClient{
			MockAgent: &MockAgent{
				ServiceFunc: func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
					return &api.AgentService{
						ID:      serviceID,
						Service: "api",
						Tags:    []string{"role-old", "static"},
					}, nil, nil
				},
				ServiceRegisterFunc: func(reg *api.AgentServiceRegistration) error {
					gotTags = reg.Tags
					return nil
				},
			},
		},
		&MockCommandExecutor{MockOutput: []byte("primary")},
		"api-1",
		"echo primary",
		time.Minute,
		"role",
		discardTagitLogger(),
	)

	if err := tagit.ReconcileOnce(); err != nil {
		t.Fatalf("ReconcileOnce() error = %v", err)
	}

	want := []string{"role-primary", "static"}
	if !sameStringSlice(gotTags, want) {
		t.Fatalf("registered tags = %v, want %v", gotTags, want)
	}
}

func TestTagIt_ReconcileOnceSurfacesScriptFailure(t *testing.T) {
	tagit := New(
		&MockConsulClient{
			MockAgent: &MockAgent{
				ServiceFunc: func(serviceID string, q *api.QueryOptions) (*api.AgentService, *api.QueryMeta, error) {
					return &api.AgentService{ID: serviceID}, nil, nil
				},
			},
		},
		&MockCommandExecutor{MockError: fmt.Errorf("script failed")},
		"api-1",
		"bad-script",
		time.Minute,
		"role",
		discardTagitLogger(),
	)

	err := tagit.ReconcileOnce()
	if err == nil {
		t.Fatal("ReconcileOnce() error = nil, want error")
	}
	if got := err.Error(); got != "error running script: script failed" {
		t.Fatalf("ReconcileOnce() error = %q, want script context", got)
	}
}

func TestScheduler_RunUsesTriggeredTicks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticks := make(chan time.Time)
	calls := make(chan struct{}, 2)
	done := make(chan struct{})
	scheduler := Scheduler{
		Interval: time.Minute,
		Ticks:    ticks,
		RunOnce: func() error {
			calls <- struct{}{}
			return nil
		},
		Logger: discardTagitLogger(),
	}

	go func() {
		defer close(done)
		scheduler.Run(ctx)
	}()

	ticks <- time.Now()
	<-calls
	ticks <- time.Now()
	<-calls

	cancel()
	<-done
}

func TestScheduler_RunLogsAndContinuesAfterError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticks := make(chan time.Time)
	calls := 0
	seenCalls := make(chan int, 2)
	done := make(chan struct{})
	scheduler := Scheduler{
		Interval: time.Minute,
		Ticks:    ticks,
		RunOnce: func() error {
			calls++
			seenCalls <- calls
			if calls == 1 {
				return fmt.Errorf("temporary")
			}
			return nil
		},
		Logger: discardTagitLogger(),
	}

	go func() {
		defer close(done)
		scheduler.Run(ctx)
	}()

	ticks <- time.Now()
	if got := <-seenCalls; got != 1 {
		t.Fatalf("first call = %d, want 1", got)
	}
	ticks <- time.Now()
	if got := <-seenCalls; got != 2 {
		t.Fatalf("second call = %d, want 2", got)
	}

	cancel()
	<-done
}

func TestScheduler_RunStopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	scheduler := Scheduler{
		Interval: time.Minute,
		Ticks:    make(chan time.Time),
		RunOnce: func() error {
			calls++
			return nil
		},
		Logger: discardTagitLogger(),
	}

	scheduler.Run(ctx)

	if calls != 0 {
		t.Fatalf("RunOnce calls = %d, want 0", calls)
	}
}

func TestScheduler_RunReturnsWhenRunOnceIsMissing(t *testing.T) {
	scheduler := Scheduler{
		Interval: time.Minute,
		Ticks:    make(chan time.Time),
	}

	scheduler.Run(t.Context())
}

func TestScheduler_RunReturnsWhenTickChannelCloses(t *testing.T) {
	ticks := make(chan time.Time)
	close(ticks)

	calls := 0
	scheduler := Scheduler{
		Interval: time.Minute,
		Ticks:    ticks,
		RunOnce: func() error {
			calls++
			return nil
		},
		Logger: discardTagitLogger(),
	}

	scheduler.Run(t.Context())

	if calls != 0 {
		t.Fatalf("RunOnce calls = %d, want 0", calls)
	}
}

func TestScheduler_RunCreatesTickerWhenTicksUnset(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	scheduler := Scheduler{
		Interval: time.Hour,
		RunOnce: func() error {
			calls++
			return nil
		},
		Logger: discardTagitLogger(),
	}

	scheduler.Run(ctx)

	if calls != 0 {
		t.Fatalf("RunOnce calls = %d, want 0", calls)
	}
}

func TestScheduler_RunRejectsInvalidGeneratedTicker(t *testing.T) {
	previous := slog.Default()
	slog.SetDefault(discardTagitLogger())
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	calls := 0
	scheduler := Scheduler{
		RunOnce: func() error {
			calls++
			return nil
		},
	}

	scheduler.Run(t.Context())

	if calls != 0 {
		t.Fatalf("RunOnce calls = %d, want 0", calls)
	}
}

func discardTagitLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
