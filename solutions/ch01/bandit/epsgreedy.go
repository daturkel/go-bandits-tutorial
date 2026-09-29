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
	if p.rng.Float64() < p.Epsilon {
		return p.rng.IntN(len(p.values))
	}
	return p.greedy()
}

// greedy returns the arm with the highest mean, breaking ties uniformly at random.
func (p *EpsilonGreedy) greedy() int {
	best, ties := 0, 1
	for i := 1; i < len(p.values); i++ {
		switch {
		case p.values[i] > p.values[best]:
			best, ties = i, 1
		case p.values[i] == p.values[best]:
			ties++
			if p.rng.IntN(ties) == 0 {
				best = i
			}
		}
	}
	return best
}

// Update folds an observed reward into the arm's running mean.
func (p *EpsilonGreedy) Update(arm int, reward float64) {
	p.counts[arm]++
	p.values[arm] += (reward - p.values[arm]) / float64(p.counts[arm])
}
