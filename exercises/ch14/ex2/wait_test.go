package ex2

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWaitHealthyReturnsWhenCheckPasses(t *testing.T) {
	calls := 0
	err := WaitHealthy(context.Background(), time.Millisecond, func(context.Context) error {
		calls++
		if calls < 4 {
			return errors.New("not yet")
		}
		return nil
	})
	if err != nil || calls != 4 {
		t.Errorf("err = %v after %d calls, want nil after 4", err, calls)
	}
}

func TestWaitHealthyChecksImmediately(t *testing.T) {
	start := time.Now()
	err := WaitHealthy(context.Background(), time.Hour, func(context.Context) error { return nil })
	if err != nil || time.Since(start) > time.Second {
		t.Errorf("err = %v after %v; a healthy service must not wait for the first tick", err, time.Since(start))
	}
}

func TestWaitHealthyGivesUpWithTheLastError(t *testing.T) {
	last := errors.New("connection refused")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	err := WaitHealthy(ctx, 5*time.Millisecond, func(context.Context) error { return last })
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, last) {
		t.Errorf("err = %v, want it to wrap both DeadlineExceeded and the last check error", err)
	}
}

func TestWaitHealthyPassesTheContextToCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- WaitHealthy(ctx, time.Millisecond, func(ctx context.Context) error {
			<-ctx.Done() // a check that hangs until told to stop
			return ctx.Err()
		})
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v, want Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("WaitHealthy did not return after the context was cancelled")
	}
}
