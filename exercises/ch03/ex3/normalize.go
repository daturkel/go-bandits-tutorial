package ex3

// Normalize scales src so its elements sum to 1 and writes the result into
// dst, reusing dst's backing array. It returns the filled slice, which has
// len(src) elements. If src sums to zero the result is all zeros.
//
// The starter allocates a new slice on every call; the test measures that
// with testing.AllocsPerRun and requires zero allocations when dst has enough
// capacity. Run the benchmark to see the difference:
//
//	go test -bench . -benchmem ./ex3
func Normalize(dst, src []float64) []float64 {
	var sum float64
	for _, x := range src {
		sum += x
	}
	out := make([]float64, len(src))
	if sum == 0 {
		return out
	}
	for i, x := range src {
		out[i] = x / sum
	}
	return out
}
