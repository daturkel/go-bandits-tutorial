package ex2

import "sync"

// Stats accumulates a count and a sum from many goroutines.
//
// It has a bug: run `go vet ./ex2` to see what the toolchain says, then fix
// it. Do not change the tests.
type Stats struct {
	mu  sync.Mutex
	n   int
	sum float64
}

// NewStats returns an empty Stats.
func NewStats() Stats {
	return Stats{}
}

// Add records one observation.
func (s Stats) Add(x float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	s.sum += x
}

// Count returns the number of observations.
func (s Stats) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.n
}

// Mean returns the average observation, or 0 if there are none.
func (s Stats) Mean() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.n == 0 {
		return 0
	}
	return s.sum / float64(s.n)
}
