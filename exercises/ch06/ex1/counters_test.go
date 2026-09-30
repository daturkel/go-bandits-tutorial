package ex1

import (
	"sync"
	"testing"
)

type counter interface {
	Inc()
	Value() int
}

func TestCountersUnderContention(t *testing.T) {
	impls := map[string]counter{
		"LockedCounter": &LockedCounter{},
		"AtomicCounter": &AtomicCounter{},
	}
	const goroutines, perGoroutine = 50, 2000
	for name, c := range impls {
		t.Run(name, func(t *testing.T) {
			var wg sync.WaitGroup
			for range goroutines {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for range perGoroutine {
						c.Inc()
					}
				}()
			}
			wg.Wait()
			if got, want := c.Value(), goroutines*perGoroutine; got != want {
				t.Errorf("Value() = %d, want %d (updates were lost)", got, want)
			}
		})
	}
}

// Value must be safe to call while other goroutines are incrementing. This
// only fails under `go test -race`.
func TestValueWhileIncrementing(t *testing.T) {
	for name, c := range map[string]counter{"LockedCounter": &LockedCounter{}, "AtomicCounter": &AtomicCounter{}} {
		t.Run(name, func(t *testing.T) {
			done := make(chan struct{})
			go func() {
				defer close(done)
				for range 1000 {
					c.Inc()
				}
			}()
			for range 1000 {
				_ = c.Value()
			}
			<-done
		})
	}
}
