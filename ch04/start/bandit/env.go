package bandit

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
)

// Env is a Bernoulli bandit: pulling arm i pays 1 with probability probs[i]
// and 0 otherwise.
type Env struct {
	probs []float64
	rng   *rand.Rand
}

// NewEnv builds an environment from per-arm success probabilities. It reports
// every invalid probability at once, not just the first.
func NewEnv(probs []float64, rng *rand.Rand) (*Env, error) {
	if len(probs) == 0 {
		return nil, ErrNoArms
	}
	var errs []error
	for arm, p := range probs {
		if !(p >= 0 && p <= 1) { // written this way so NaN is rejected too
			errs = append(errs, &ArmError{Arm: arm, Err: fmt.Errorf("%w: got %v", ErrInvalidProbability, p)})
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("invalid environment: %w", err)
	}
	return &Env{probs: slices.Clone(probs), rng: rng}, nil
}

// NumArms reports how many arms the environment has.
func (e *Env) NumArms() int { return len(e.probs) }

// Prob returns the true success probability of an arm.
func (e *Env) Prob(arm int) float64 { return e.probs[arm] }

// Pull plays an arm and returns the observed reward, 0 or 1.
func (e *Env) Pull(arm int) float64 {
	if e.rng.Float64() < e.probs[arm] {
		return 1
	}
	return 0
}

// Best returns the arm with the highest success probability, and that probability.
func (e *Env) Best() (arm int, p float64) {
	for i, q := range e.probs {
		if i == 0 || q > p {
			arm, p = i, q
		}
	}
	return arm, p
}
