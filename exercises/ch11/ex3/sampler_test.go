package ex3

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestSamplerPattern(t *testing.T) {
	s := NewSampler(3)
	var got []bool
	for range 10 {
		got = append(got, s.Allow())
	}
	want := []bool{true, false, false, true, false, false, true, false, false, true}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("call %d: got %v, want %v (all: %v)", i+1, got[i], want[i], got)
		}
	}
}

func TestSamplerOneAndNonPositive(t *testing.T) {
	for _, n := range []int{1, 0, -5} {
		s := NewSampler(n)
		for i := range 5 {
			if !s.Allow() {
				t.Fatalf("NewSampler(%d): call %d was refused, want every call allowed", n, i+1)
			}
		}
	}
}

// Under concurrency exactly one call in n must be let through, no more, no fewer.
func TestSamplerConcurrent(t *testing.T) {
	const n, goroutines, perGoroutine = 10, 20, 500
	s := NewSampler(n)
	var allowed atomic.Int64
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perGoroutine {
				if s.Allow() {
					allowed.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	// 10,000 calls: allowed are calls 1, 11, 21, ..., 9991, i.e. 1000 of them.
	if got := allowed.Load(); got != goroutines*perGoroutine/n {
		t.Errorf("%d calls allowed, want %d", got, goroutines*perGoroutine/n)
	}
}
