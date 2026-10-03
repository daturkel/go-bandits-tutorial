package extras

// ParseProbs parses a comma-separated list such as "0.2,0.5,0.8".
//
// Errors must say which entry is wrong, e.g. "entry 2: ...", and must wrap the
// underlying cause with %w so callers can use errors.Is and errors.As:
//   - text that is not a number: wrap the *strconv.NumError from ParseFloat
//   - a number outside [0, 1]: wrap bandit.ErrInvalidProbability
//
// Return the first problem found. An empty string is an error too, but it
// does not need to wrap anything in particular.
func ParseProbs(s string) ([]float64, error) {
	// TASK E1
	return nil, nil
}

// Normalize scales src so its elements sum to 1 and writes the result into
// dst, reusing dst's backing array. It returns the filled slice, which has
// len(src) elements. If src sums to zero the result is all zeros.
//
// The test measures allocations with testing.AllocsPerRun and requires zero
// when dst has enough capacity.
func Normalize(dst, src []float64) []float64 {
	// TASK E2
	return nil
}
