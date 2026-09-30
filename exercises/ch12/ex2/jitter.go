package ex2

import (
	"math/rand/v2"
	"time"
)

// Jitter returns interval randomly shifted by up to frac of itself in either
// direction: a value in [interval*(1-frac), interval*(1+frac)]. Replicas that
// all start together and sync every 10 seconds would otherwise hit the
// database in the same instant, every time.
//
// frac is clamped to [0, 1]. With frac 0 the result is exactly interval. The
// randomness comes from r, so tests can seed it.
func Jitter(interval time.Duration, frac float64, r *rand.Rand) time.Duration {
	// TODO
	return 0
}
