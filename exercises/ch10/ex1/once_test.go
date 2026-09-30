package ex1

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func newClock() *fakeClock { return &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)} }

func TestRunsOncePerKey(t *testing.T) {
	clk := newClock()
	o := NewOnce(time.Minute, clk.Now)
	var runs int
	f := func() error { runs++; return nil }

	if ran, err := o.Do("a", f); !ran || err != nil {
		t.Fatalf("first Do = (%v, %v), want (true, nil)", ran, err)
	}
	if ran, err := o.Do("a", f); ran || err != nil {
		t.Fatalf("second Do = (%v, %v), want (false, nil)", ran, err)
	}
	if ran, _ := o.Do("b", f); !ran {
		t.Error("a different key must run independently")
	}
	if runs != 2 {
		t.Errorf("f ran %d times, want 2", runs)
	}
}

func TestKeyFreesAfterTTL(t *testing.T) {
	clk := newClock()
	o := NewOnce(time.Minute, clk.Now)
	f := func() error { return nil }
	o.Do("a", f)
	clk.Advance(59 * time.Second)
	if ran, _ := o.Do("a", f); ran {
		t.Error("ran again before the ttl passed")
	}
	clk.Advance(2 * time.Second)
	if ran, _ := o.Do("a", f); !ran {
		t.Error("did not run again after the ttl passed")
	}
}

func TestFailureReleasesTheClaim(t *testing.T) {
	o := NewOnce(time.Minute, newClock().Now)
	boom := errors.New("boom")
	ran, err := o.Do("a", func() error { return boom })
	if !ran || !errors.Is(err, boom) {
		t.Fatalf("Do = (%v, %v), want (true, boom)", ran, err)
	}
	if ran, err := o.Do("a", func() error { return nil }); !ran || err != nil {
		t.Errorf("retry after failure = (%v, %v), want it to run", ran, err)
	}
}

func TestConcurrentCallsRunOnce(t *testing.T) {
	o := NewOnce(time.Minute, newClock().Now)
	var runs, skipped atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ran, _ := o.Do("same", func() error {
				runs.Add(1)
				time.Sleep(20 * time.Millisecond) // a slow action; the others must not wait for it
				return nil
			})
			if !ran {
				skipped.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if runs.Load() != 1 || skipped.Load() != 49 {
		t.Errorf("f ran %d times and %d calls were skipped, want 1 and 49", runs.Load(), skipped.Load())
	}
}
