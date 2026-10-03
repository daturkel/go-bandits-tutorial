package ex3

import (
	"sync"
	"testing"
)

type pair struct{ a, b int }

func TestCellLoadStore(t *testing.T) {
	var c Cell[pair]
	if got := c.Load(); got != (pair{}) {
		t.Errorf("zero Cell loaded %v, want the zero value", got)
	}
	c.Store(pair{1, 2})
	if got := c.Load(); got != (pair{1, 2}) {
		t.Errorf("Load = %v, want {1 2}", got)
	}
}

func TestCellUpdateReturnsNewValue(t *testing.T) {
	var c Cell[int]
	c.Store(10)
	if got := c.Update(func(v int) int { return v + 5 }); got != 15 {
		t.Errorf("Update returned %d, want 15", got)
	}
	if got := c.Load(); got != 15 {
		t.Errorf("Load after Update = %d, want 15", got)
	}
}

func TestCellConcurrentUpdates(t *testing.T) {
	var c Cell[pair]
	const goroutines, perGoroutine = 20, 1000
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perGoroutine {
				c.Update(func(p pair) pair {
					p.a++
					p.b += 2
					return p
				})
			}
		}()
	}
	wg.Wait()
	want := pair{goroutines * perGoroutine, 2 * goroutines * perGoroutine}
	if got := c.Load(); got != want {
		t.Errorf("final value %v, want %v (updates were lost)", got, want)
	}
}

// A deterministic version of the race: the first Update pauses after reading
// the value, a second Update completes in the meantime, and then the first
// one carries on. A compare-and-swap loop notices and retries with the new
// value; a read-then-write overwrites the second update.
func TestCellUpdateRetriesAfterAConflict(t *testing.T) {
	var c Cell[int]
	c.Store(0)
	read, resume, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		first := true
		c.Update(func(v int) int {
			if first {
				first = false
				close(read)
				<-resume
			}
			return v + 1
		})
	}()
	<-read
	c.Update(func(v int) int { return v + 10 })
	close(resume)
	<-done
	if got := c.Load(); got != 11 {
		t.Errorf("final value %d, want 11: the update made while the first one was paused was lost", got)
	}
}
