package ex3

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

// recorder is a fake sleep that records the delays it was asked for.
type recorder struct{ delays []time.Duration }

func (r *recorder) sleep(ctx context.Context, d time.Duration) error {
	r.delays = append(r.delays, d)
	return ctx.Err()
}

func TestRetrySucceedsEventually(t *testing.T) {
	var r recorder
	calls := 0
	err := Retry(context.Background(), 5, 100*time.Millisecond, time.Second, r.sleep,
		func(ctx context.Context, attempt int) error {
			calls++
			if attempt != calls {
				t.Errorf("attempt = %d on call %d", attempt, calls)
			}
			if attempt < 3 {
				return errors.New("not yet")
			}
			return nil
		})
	if err != nil || calls != 3 {
		t.Fatalf("err = %v after %d calls, want nil after 3", err, calls)
	}
	if want := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond}; !slices.Equal(r.delays, want) {
		t.Errorf("delays = %v, want %v", r.delays, want)
	}
}

func TestRetryBackoffIsCapped(t *testing.T) {
	var r recorder
	boom := errors.New("down")
	err := Retry(context.Background(), 6, 100*time.Millisecond, 350*time.Millisecond, r.sleep,
		func(context.Context, int) error { return boom })
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the last attempt's error", err)
	}
	want := []time.Duration{100, 200, 350, 350, 350}
	for i := range want {
		want[i] *= time.Millisecond
	}
	if !slices.Equal(r.delays, want) {
		t.Errorf("delays = %v, want %v (5 sleeps for 6 attempts, none after the last)", r.delays, want)
	}
}

func TestRetryStopsOnPermanentError(t *testing.T) {
	var r recorder
	bad := errors.New("password authentication failed")
	calls := 0
	err := Retry(context.Background(), 5, time.Millisecond, time.Second, r.sleep,
		func(context.Context, int) error {
			calls++
			return Permanent(bad)
		})
	if !errors.Is(err, bad) || calls != 1 || len(r.delays) != 0 {
		t.Errorf("err = %v, calls = %d, sleeps = %v; want the error after 1 call and no sleeping", err, calls, r.delays)
	}
}

func TestRetryStopsWhenContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := Retry(ctx, 10, time.Millisecond, time.Second,
		func(ctx context.Context, d time.Duration) error {
			cancel() // the caller gives up while we wait
			return ctx.Err()
		},
		func(context.Context, int) error {
			calls++
			return errors.New("down")
		})
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Errorf("err = %v after %d calls, want context.Canceled after 1", err, calls)
	}
}

func TestRetryZeroAttempts(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), 0, time.Millisecond, time.Second, (&recorder{}).sleep,
		func(context.Context, int) error { calls++; return nil })
	if err == nil || calls != 0 {
		t.Errorf("Retry with 0 attempts = %v after %d calls, want an error and no calls", err, calls)
	}
}

func TestSleepContext(t *testing.T) {
	if err := SleepContext(context.Background(), 5*time.Millisecond); err != nil {
		t.Errorf("full sleep = %v, want nil", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if err := SleepContext(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled sleep = %v, want context.Canceled", err)
	}
	if time.Since(start) > time.Second {
		t.Error("SleepContext did not return promptly when the context was cancelled")
	}
}
