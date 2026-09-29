package ex1

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

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
	if s == "" {
		return nil, errors.New("no probabilities given")
	}
	fields := strings.Split(s, ",")
	probs := make([]float64, 0, len(fields))
	for i, f := range fields {
		p, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
		if err != nil {
			return nil, fmt.Errorf("entry %d: %w", i, err)
		}
		if p < 0 || p > 1 {
			return nil, fmt.Errorf("entry %d: %w: %v", i, ErrOutOfRange, p)
		}
		probs = append(probs, p)
	}
	return probs, nil
}
