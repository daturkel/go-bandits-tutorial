package bandit

import "math/rand/v2"

// EpsilonGreedy explores a random arm with probability Epsilon and otherwise
// exploits the arm with the best average reward so far.
type EpsilonGreedy struct {
	Epsilon float64

	counts []int     // pulls per arm
	values []float64 // running mean reward per arm
	rng    *rand.Rand
}

// NewEpsilonGreedy returns a policy for nArms arms.
func NewEpsilonGreedy(nArms int, epsilon float64, rng *rand.Rand) *EpsilonGreedy {
	return &EpsilonGreedy{
		Epsilon: epsilon,
		counts:  make([]int, nArms),
		values:  make([]float64, nArms),
		rng:     rng,
	}
}

// Select chooses the next arm to pull.
func (p *EpsilonGreedy) Select() int {
	// TASK 5: with probability p.Epsilon return a random arm, otherwise the
	// greedy one.
	return 0
}

// greedy returns the arm with the highest mean, breaking ties uniformly at random.
func (p *EpsilonGreedy) greedy() int {
	// TASK 4: return the index of the largest value in p.values.
	// When several arms tie, pick one of them at random using p.rng.IntN(n),
	// which returns a number from 0 to n-1. Picking the first would bias
	// the very first pulls, when every mean is 0.
	return 0
}

// Update folds an observed reward into the arm's running mean.
func (p *EpsilonGreedy) Update(arm int, reward float64) {
	// TASK 3: count the pull, then move the mean toward the reward:
	//   mean += (reward - mean) / count
}
