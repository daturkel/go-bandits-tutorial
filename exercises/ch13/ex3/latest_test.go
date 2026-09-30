package ex3

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestWaitReturnsImmediatelyWhenAlreadyNewer(t *testing.T) {
	l := NewLatest("a")
	l.Set("b")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	v, ver, err := l.Wait(ctx, 0)
	if err != nil || v != "b" || ver != 1 {
		t.Errorf("Wait(0) = %q, %d, %v; want b, 1, nil", v, ver, err)
	}
}

func TestWaitBlocksUntilSet(t *testing.T) {
	l := NewLatest(10)
	type result struct {
		v   int
		ver uint64
		err error
	}
	got := make(chan result, 1)
	go func() {
		v, ver, err := l.Wait(context.Background(), 0)
		got <- result{v, ver, err}
	}()
	select {
	case r := <-got:
		t.Fatalf("Wait returned %v before any Set", r)
	case <-time.After(50 * time.Millisecond):
	}
	l.Set(11)
	select {
	case r := <-got:
		if r.err != nil || r.v != 11 || r.ver != 1 {
			t.Errorf("Wait = %+v, want 11 at version 1", r)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not wake up")
	}
}

func TestWaitHonoursContext(t *testing.T) {
	l := NewLatest(0)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, _, err := l.Wait(ctx, 0)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want DeadlineExceeded", err)
	}
}

func TestEveryWaiterWakes(t *testing.T) {
	l := NewLatest(0)
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if v, _, err := l.Wait(ctx, 0); err != nil || v != 5 {
				t.Errorf("waiter got %d, %v", v, err)
			}
		}()
	}
	time.Sleep(50 * time.Millisecond)
	l.Set(5)
	wg.Wait()
}

// A follower that keeps the version it last saw never misses the latest state,
// however fast the writer is (run with -race).
func TestFollowerCatchesUp(t *testing.T) {
	l := NewLatest(0)
	const last = 1000
	done := make(chan int, 1)
	go func() {
		var seen uint64
		var v int
		for v != last {
			var err error
			if v, seen, err = l.Wait(context.Background(), seen); err != nil {
				return
			}
		}
		done <- v
	}()
	for i := 1; i <= last; i++ {
		l.Set(i)
	}
	select {
	case v := <-done:
		if v != last {
			t.Errorf("follower ended at %d", v)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("follower never saw the final value")
	}
}
