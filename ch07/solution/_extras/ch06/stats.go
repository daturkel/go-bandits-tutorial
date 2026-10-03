package extras

import "sync"

// Stats accumulates a count and a sum from many goroutines.
//
// A struct containing a mutex must not be copied. With value receivers every
// call locked and updated a private copy, so the caller's Stats never changed.
// Pointer receivers, and a constructor that returns a pointer, fix it.
type Stats struct {
	mu  sync.Mutex
	n   int
	sum float64
}

// NewStats returns an empty Stats.
func NewStats() *Stats {
	return &Stats{}
}

// Add records one observation.
func (s *Stats) Add(x float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	s.sum += x
}

// Count returns the number of observations.
func (s *Stats) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.n
}

// Mean returns the average observation, or 0 if there are none.
func (s *Stats) Mean() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.n == 0 {
		return 0
	}
	return s.sum / float64(s.n)
}
