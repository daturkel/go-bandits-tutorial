package ex3

import "sync/atomic"

// Sampler lets through the first call and then every nth call after it, so a
// hot code path can log "1 in n" events. It must be safe for concurrent use
// without a mutex (use a sync/atomic counter), and its zero value is not
// used: create one with NewSampler.
//
//	s := NewSampler(100)
//	if s.Allow() { log.Info("request", ...) }   // calls 1, 101, 201, ...
//
// If n is less than 1, Allow is always true.
type Sampler struct {
	n     uint64
	calls atomic.Uint64
}

// NewSampler returns a Sampler that allows one call in n.
func NewSampler(n int) *Sampler {
	return &Sampler{n: uint64(max(n, 1))}
}

// Allow reports whether this call should be let through.
func (s *Sampler) Allow() bool {
	// Add returns the new value, so the first call sees 1; subtracting one
	// makes the sequence 0, 1, 2, ... and "every nth starting with the first"
	// is "index % n == 0".
	return (s.calls.Add(1)-1)%s.n == 0
}
