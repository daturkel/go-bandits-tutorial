package pool

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"sync/atomic"
	"testing"
	"time"
)

func square(_ context.Context, x int) (int, error) { return x * x, nil }

func TestRunKeepsJobOrder(t *testing.T) {
	jobs := []int{5, 3, 8, 1, 9, 2}
	got, err := Run(context.Background(), Options{Workers: 3}, jobs, func(_ context.Context, x int) (int, error) {
		time.Sleep(time.Duration(10-x) * time.Millisecond) // bigger inputs finish first
		return x * x, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{25, 9, 64, 1, 81, 4}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRunEdgeCases(t *testing.T) {
	got, err := Run(context.Background(), Options{}, []int(nil), square)
	if err != nil || len(got) != 0 {
		t.Errorf("no jobs: got (%v, %v), want empty and nil", got, err)
	}
	got, err = Run(context.Background(), Options{Workers: 50}, []int{2, 3}, square)
	if err != nil || !slices.Equal(got, []int{4, 9}) {
		t.Errorf("more workers than jobs: got (%v, %v)", got, err)
	}
	got, err = Run(context.Background(), Options{Workers: -1}, []int{4}, square)
	if err != nil || !slices.Equal(got, []int{16}) {
		t.Errorf("negative workers: got (%v, %v)", got, err)
	}
}

// Run never has more than Workers calls to f in flight, but does use them all.
func TestRunBoundsConcurrency(t *testing.T) {
	const workers = 3
	var inFlight, peak atomic.Int32
	jobs := make([]int, 30)
	_, err := Run(context.Background(), Options{Workers: workers}, jobs, func(_ context.Context, _ int) (int, error) {
		n := inFlight.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		inFlight.Add(-1)
		return 0, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := peak.Load(); got != workers {
		t.Errorf("peak concurrency %d, want exactly %d", got, workers)
	}
}

func TestRunOnDone(t *testing.T) {
	var seen []int
	opts := Options{Workers: 4, OnDone: func(done, total int) {
		if total != 20 {
			t.Errorf("total = %d, want 20", total)
		}
		seen = append(seen, done) // no lock needed: calls never overlap
	}}
	if _, err := Run(context.Background(), opts, make([]int, 20), square); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 20 || seen[0] != 1 || seen[19] != 20 || !slices.IsSorted(seen) {
		t.Errorf("OnDone saw %v, want 1..20 in order", seen)
	}
}

func TestRunFirstErrorCancelsTheRest(t *testing.T) {
	before := runtime.NumGoroutine()
	boom := errors.New("boom")
	var started atomic.Int32

	_, err := Run(context.Background(), Options{Workers: 2}, make([]int, 1000), func(ctx context.Context, _ int) (int, error) {
		n := started.Add(1)
		if n == 3 {
			return 0, boom
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(20 * time.Millisecond):
			return 0, nil
		}
	})
	if !errors.Is(err, boom) {
		t.Fatalf("error = %v, want boom (not the cancellations it caused)", err)
	}
	if got := started.Load(); got > 20 {
		t.Errorf("%d jobs started after the failure; the rest should be skipped", got)
	}
	waitForGoroutines(t, before)
}

func TestRunReturnsLowestIndexedError(t *testing.T) {
	jobs := []int{0, 1, 2, 3}
	for range 20 {
		_, err := Run(context.Background(), Options{Workers: 4}, jobs, func(_ context.Context, i int) (int, error) {
			if i >= 1 {
				return 0, errors.New("failed job " + string(rune('0'+i)))
			}
			return 0, nil
		})
		// All four start together, so jobs 1-3 all fail; job 1 must be reported.
		if err == nil || err.Error() != "failed job 1" {
			t.Fatalf("error = %v, want \"failed job 1\"", err)
		}
	}
}

func TestRunCallerCancellation(t *testing.T) {
	before := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	release := make(chan struct{})
	go func() {
		<-release
		cancel()
	}()

	_, err := Run(ctx, Options{Workers: 2}, make([]int, 100), func(ctx context.Context, _ int) (int, error) {
		select {
		case release <- struct{}{}:
		default:
		}
		<-ctx.Done()
		return 0, ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	waitForGoroutines(t, before)
}

func TestRunAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var calls atomic.Int32
	_, err := Run(ctx, Options{}, make([]int, 10), func(ctx context.Context, _ int) (int, error) {
		calls.Add(1)
		return 0, ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

// waitForGoroutines fails the test if goroutines started since `before` do
// not exit. It polls, because exiting goroutines need a moment to be noticed.
func waitForGoroutines(t *testing.T, before int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<16)
			t.Fatalf("goroutine leak: %d running, %d before\n%s", runtime.NumGoroutine(), before, buf[:runtime.Stack(buf, true)])
		}
		time.Sleep(5 * time.Millisecond)
	}
}
