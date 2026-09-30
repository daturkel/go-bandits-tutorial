package bandit

import (
	"math/rand/v2"
)

// Argmax returns the index of the largest element of xs. Ties go to the lowest
// index, or to a uniformly random one among the tied indices when rng is
// non-nil. It panics on an empty slice, like slices.Max.
//
// The type parameter T is constrained to cmp.Ordered, so the same function
// serves []float64 estimates and []int counts.
func Argmax(xs []float64, rng *rand.Rand) int {
	// TASK 1: make this generic. Give it a type parameter T constrained to
	// cmp.Ordered, take xs []T, and compare elements with cmp.Compare (the
	// < operator is not available on a bare type parameter). Then write the
	// body as the comment above describes.
	return 0
}
