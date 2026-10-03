package extras

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"
)

func TestFetchReturnsResult(t *testing.T) {
	got, err := Fetch(context.Background(), func() string { return "hello" })
	if err != nil || got != "hello" {
		t.Errorf("Fetch = (%q, %v), want (hello, nil)", got, err)
	}
}

func TestFetchHonoursContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := Fetch(ctx, func() string {
		time.Sleep(2 * time.Second)
		return "late"
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want context.DeadlineExceeded", err)
	}
	if time.Since(start) > time.Second {
		t.Error("Fetch waited for the work instead of returning when the context ended")
	}
}

func TestFetchDoesNotLeakGoroutines(t *testing.T) {
	before := runtime.NumGoroutine()
	for range 100 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		Fetch(ctx, func() string {
			time.Sleep(20 * time.Millisecond)
			return "late"
		})
		cancel()
	}
	// Give the abandoned work time to finish, then look for stragglers.
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before {
		if time.Now().After(deadline) {
			t.Fatalf("%d goroutines still running, %d before: Fetch leaks", runtime.NumGoroutine(), before)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
