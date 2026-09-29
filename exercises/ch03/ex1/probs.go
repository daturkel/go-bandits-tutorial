package ex1

import "errors"

// ErrOutOfRange is returned (wrapped) when a probability is not in [0, 1].
var ErrOutOfRange = errors.New("probability out of range")

// ParseProbs parses a comma-separated list such as "0.2,0.5,0.8".
//
// Errors must say which entry is wrong, e.g. "entry 2: ...", and must wrap the
// underlying cause with %w so callers can use errors.Is and errors.As:
//   - text that is not a number: wrap the *strconv.NumError from ParseFloat
//   - a number outside [0, 1]: wrap ErrOutOfRange
//
// Return the first problem found. An empty string is an error too, but it
// does not need to wrap anything in particular.
func ParseProbs(s string) ([]float64, error) {
	// TODO
	return nil, nil
}
