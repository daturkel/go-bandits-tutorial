// Package stat holds the small numeric helpers the harness needs.
package stat

import "math"

// Number is the set of types the helpers accept.
//
// TASK 5: this only allows float64 for now. Widen it to a union that also
// takes int, int64 and float32, and let it accept named types built on them
// (type Reward float64): that is what the ~ operator is for.
type Number interface {
	~float64
}

// Mean returns the arithmetic mean, or NaN for an empty slice.
func Mean[T Number](xs []T) float64 {
	// TASK 5: NaN for an empty slice. Otherwise sum as float64 (convert each
	// element with float64(x)) and divide by the count.
	return 0
}

// StdErr returns the standard error of the mean (sample standard deviation
// over sqrt(n)), or 0 when there are fewer than two samples.
func StdErr[T Number](xs []T) float64 {
	n := len(xs)
	if n < 2 {
		return 0
	}
	m := Mean(xs)
	var ss float64
	for _, x := range xs {
		d := float64(x) - m
		ss += d * d
	}
	return math.Sqrt(ss/float64(n-1)) / math.Sqrt(float64(n))
}

// Sample picks at most n evenly spaced elements of xs, always including the
// first and the last. It works on a slice of anything, so T is unconstrained.
func Sample[T any](xs []T, n int) []T {
	if n >= len(xs) || len(xs) == 0 {
		return append([]T(nil), xs...)
	}
	if n < 2 {
		return []T{xs[len(xs)-1]}
	}
	out := make([]T, 0, n)
	for i := range n {
		out = append(out, xs[i*(len(xs)-1)/(n-1)])
	}
	return out
}
