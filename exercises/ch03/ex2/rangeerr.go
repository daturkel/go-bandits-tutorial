package ex2

import "errors"

// ErrOutOfRange is the sentinel that every *RangeError unwraps to.
var ErrOutOfRange = errors.New("arm out of range")

// RangeError says which arm was requested and how many exist.
//
// Its Error method must produce: arm 7 out of range [0, 3)
// and it must unwrap to ErrOutOfRange so that errors.Is(err, ErrOutOfRange)
// is true for any error containing a *RangeError.
type RangeError struct {
	Arm, N int
}

func (e *RangeError) Error() string {
	// TODO
	return ""
}

// Unwrap connects the typed error to the sentinel.
func (e *RangeError) Unwrap() error {
	// TODO
	return nil
}

// CheckArm returns nil when 0 <= arm < n and a *RangeError otherwise.
func CheckArm(arm, n int) error {
	// TODO
	return nil
}
