package bandit

import (
	"errors"
	"fmt"
)

// Sentinel errors: callers test for them with errors.Is, however many layers
// of context have been wrapped around them.
var (
	ErrNoArms             = errors.New("at least one arm is required")
	ErrInvalidProbability = errors.New("probability must be between 0 and 1")
	ErrInvalidEpsilon     = errors.New("epsilon must be between 0 and 1")
	ErrUnknownScenario    = errors.New("unknown scenario")
	ErrUnknownPolicy      = errors.New("unknown policy")
	ErrArmOutOfRange      = errors.New("arm out of range")
)

// ArmError attaches an arm index to an underlying error. Callers reach it
// with errors.As when they need the index, and see through it with errors.Is.
type ArmError struct {
	Arm int
	Err error
}

func (e *ArmError) Error() string { return fmt.Sprintf("arm %d: %v", e.Arm, e.Err) }

// Unwrap lets errors.Is and errors.As look inside.
func (e *ArmError) Unwrap() error { return e.Err }
