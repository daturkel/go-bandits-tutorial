package ex1

import "cmp"

// Number is the set of types SumBy can add up.
type Number interface {
	~int | ~int64 | ~float32 | ~float64
}

// SumBy returns the sum of f(x) over all x in xs. The sum of an empty slice is
// the zero value of N.
//
// Both T and N are type parameters: T is what the slice holds, N is what f
// produces. Write the body, then think about whether N could be inferred at a
// call site (it can; look at the tests).
func SumBy[T any, N Number](xs []T, f func(T) N) N {
	var sum N
	for _, x := range xs {
		sum += f(x)
	}
	return sum
}

// MaxBy returns the element of xs for which key is largest, and true. For an
// empty slice it returns the zero value of T and false. When several elements
// tie, the first one wins.
func MaxBy[T any, K cmp.Ordered](xs []T, key func(T) K) (T, bool) {
	var best T
	if len(xs) == 0 {
		return best, false
	}
	best, bestKey := xs[0], key(xs[0])
	for _, x := range xs[1:] {
		if k := key(x); k > bestKey {
			best, bestKey = x, k
		}
	}
	return best, true
}
