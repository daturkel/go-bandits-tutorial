package extras

import (
	"math"
	"sync"
	"testing"
)

func TestStats(t *testing.T) {
	s := NewStats()
	var wg sync.WaitGroup
	for i := 1; i <= 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Add(float64(i))
		}()
	}
	wg.Wait()

	if got := s.Count(); got != 100 {
		t.Errorf("Count() = %d, want 100", got)
	}
	if got := s.Mean(); math.Abs(got-50.5) > 1e-9 {
		t.Errorf("Mean() = %v, want 50.5", got)
	}
}

func TestStatsEmpty(t *testing.T) {
	s := NewStats()
	if s.Count() != 0 || s.Mean() != 0 {
		t.Errorf("empty Stats: count %d mean %v", s.Count(), s.Mean())
	}
}
