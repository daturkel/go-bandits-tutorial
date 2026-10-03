package extras

import (
	"cmp"

	"banditlab/stat"
)

// SumBy returns the sum of f(x) over all x in xs. The sum of an empty slice is
// the zero value of N.
//
// Both T and N are type parameters: T is what the slice holds, N is what f
// produces. Write the body, then think about whether N could be inferred at a
// call site (it can; look at the tests).
func SumBy[T any, N stat.Number](xs []T, f func(T) N) N {
	// TASK E1
	var zero N
	return zero
}

// MaxBy returns the element of xs for which key is largest, and true. For an
// empty slice it returns the zero value of T and false. When several elements
// tie, the first one wins.
func MaxBy[T any, K cmp.Ordered](xs []T, key func(T) K) (T, bool) {
	// TASK E1
	var zero T
	return zero, false
}

// Quantile returns the q-quantile (0 <= q <= 1) of xs using linear
// interpolation between closest ranks: sort a copy of the data, compute the
// position q*(n-1), and interpolate between the values on either side.
//
//	Quantile([]float64{1, 2, 3, 4}, 0.5) == 2.5
//	Quantile([]float64{1, 2, 3, 4}, 0)   == 1
//	Quantile([]float64{1, 2, 3, 4}, 1)   == 4
//
// It must not modify xs. For an empty slice, or q outside [0, 1], return NaN.
func Quantile(xs []float64, q float64) float64 {
	// TASK E2
	return 0
}

// Counter counts how often each key has been seen. Its zero value is ready to
// use: `var c Counter[string]; c.Inc("a")` must work without a constructor.
type Counter[K comparable] struct {
	counts map[K]int
}

// Inc adds one to key's count.
func (c *Counter[K]) Inc(key K) {
	// TASK E3
}

// Get returns key's count, or 0 if it was never seen.
func (c *Counter[K]) Get(key K) int {
	// TASK E3
	return 0
}

// MostCommon returns the key with the highest count and that count. The bool
// is false when nothing has been counted. (Ties may return either key.)
func (c *Counter[K]) MostCommon() (K, int, bool) {
	// TASK E3
	var zero K
	return zero, 0, false
}
