package ex1

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// sleeper returns v after d, or ctx's error if the context ends first.
func sleeper(d time.Duration, v string, returned *atomic.Int32) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		defer returned.Add(1)
		select {
		case <-time.After(d):
			return v, nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

func TestFirstReturnsFastestAndCancelsOthers(t *testing.T) {
	var returned atomic.Int32
	start := time.Now()
	got, err := First(context.Background(),
		sleeper(5*time.Second, "slow", &returned),
		sleeper(10*time.Millisecond, "fast", &returned),
		sleeper(5*time.Second, "also slow", &returned),
	)
	if err != nil || got != "fast" {
		t.Fatalf("First = (%q, %v), want (fast, nil)", got, err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("took %v: the slow functions were not cancelled", elapsed)
	}
	if n := returned.Load(); n != 3 {
		t.Errorf("only %d of 3 functions had returned when First returned; wait for all of them", n)
	}
}

func TestFirstIgnoresEarlyFailures(t *testing.T) {
	failing := func(context.Context) (string, error) { return "", errors.New("nope") }
	var returned atomic.Int32
	got, err := First(context.Background(), failing, sleeper(20*time.Millisecond, "ok", &returned))
	if err != nil || got != "ok" {
		t.Errorf("First = (%q, %v), want (ok, nil)", got, err)
	}
}

func TestFirstAllFail(t *testing.T) {
	e1, e2 := errors.New("first failure"), errors.New("second failure")
	_, err := First(context.Background(),
		func(context.Context) (int, error) { return 0, e1 },
		func(context.Context) (int, error) { return 0, e2 },
	)
	if !errors.Is(err, e1) || !errors.Is(err, e2) {
		t.Errorf("error %v should wrap both failures", err)
	}
}

func TestFirstNoFunctions(t *testing.T) {
	if _, err := First[int](context.Background()); err == nil {
		t.Error("expected an error with no functions")
	}
}

func TestFirstCallerCancels(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	var returned atomic.Int32
	_, err := First(ctx, sleeper(5*time.Second, "slow", &returned))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want one wrapping context.DeadlineExceeded", err)
	}
}
