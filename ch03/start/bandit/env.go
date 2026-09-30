package bandit

import (
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
	// TASK 2: reject invalid input.
	//  - no arms at all: return ErrNoArms.
	//  - a probability outside [0, 1] (NaN counts as outside): describe it as
	//    an *ArmError for that arm wrapping ErrInvalidProbability, collect one
	//    for every bad arm, join them with errors.Join, and return the join
	//    wrapped with the context "invalid environment".
	// You will need the "errors" and "fmt" packages.
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
