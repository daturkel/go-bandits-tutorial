package bandit

import (
	"cmp"
	"math/rand/v2"
)

// Argmax returns the index of the largest element of xs. Ties go to the lowest
// index, or to a uniformly random one among the tied indices when rng is
// non-nil. It panics on an empty slice, like slices.Max.
//
// The type parameter T is constrained to cmp.Ordered, so the same function
// serves []float64 estimates and []int counts.
func Argmax[T cmp.Ordered](xs []T, rng *rand.Rand) int {
	if len(xs) == 0 {
		panic("bandit.Argmax: empty slice")
	}
	best, ties := 0, 1
	for i := 1; i < len(xs); i++ {
		switch c := cmp.Compare(xs[i], xs[best]); {
		case c > 0:
			best, ties = i, 1
		case c == 0 && rng != nil:
			ties++
			if rng.IntN(ties) == 0 {
				best = i
			}
		}
	}
	return best
}
