package ex2

import (
	"fmt"
	"strconv"
)

// ParseProbability turns text such as "0.25" into a number. It returns an
// error if the text is not a number, or if the number is outside 0 to 1
// (both ends allowed). Use strconv.ParseFloat. Note that it accepts the text
// "NaN" ("not a number"), which is not a probability either.
func ParseProbability(s string) (float64, error) {
	p, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("probability %q: %w", s, err)
	}
	// Written as "not inside the range" so that NaN, which compares false
	// with everything, is rejected too.
	if !(p >= 0 && p <= 1) {
		return 0, fmt.Errorf("probability %v is not between 0 and 1", p)
	}
	return p, nil
}
