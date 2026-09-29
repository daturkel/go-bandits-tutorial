package ex2

import (
	"math"
	"slices"
)

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
	if len(xs) == 0 || !(q >= 0 && q <= 1) {
		return math.NaN()
	}
	sorted := slices.Clone(xs)
	slices.Sort(sorted)
	pos := q * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	frac := pos - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}
