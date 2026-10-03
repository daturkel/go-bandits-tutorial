package bandit

import "math/rand/v2"

// Env is a Bernoulli bandit: pulling arm i pays 1 with probability probs[i]
// and 0 otherwise.
type Env struct {
	// TASK 2: the fields you need. Two are enough.
}

// NewEnv builds an environment from per-arm success probabilities.
func NewEnv(probs []float64, rng *rand.Rand) *Env {
	// TASK 2: keep the probabilities and the generator. The caller must not be
	// able to change the environment by changing its own slice later.
	return &Env{}
}

// NumArms reports how many arms the environment has.
func (e *Env) NumArms() int {
	// TASK 2
	return 0
}

// Prob returns the true success probability of an arm.
func (e *Env) Prob(arm int) float64 {
	// TASK 2
	return 0
}

// Pull plays an arm and returns the observed reward, 0 or 1.
func (e *Env) Pull(arm int) float64 {
	// TASK 3: return 1 with probability Prob(arm), otherwise 0.
	// A random number from rng.Float64() is in [0, 1).
	return 0
}

// Best returns the arm with the highest success probability, and that probability.
func (e *Env) Best() (arm int, p float64) {
	// TASK 4: loop over the probabilities and remember the largest. Because
	// arm and p are named results, they already exist here, set to zero.
	return arm, p
}
