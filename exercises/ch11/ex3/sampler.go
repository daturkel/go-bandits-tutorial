package ex3

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
	// TODO: add the state you need.
}

// NewSampler returns a Sampler that allows one call in n.
func NewSampler(n int) *Sampler {
	return &Sampler{}
}

// Allow reports whether this call should be let through.
func (s *Sampler) Allow() bool {
	// TODO
	return false
}
