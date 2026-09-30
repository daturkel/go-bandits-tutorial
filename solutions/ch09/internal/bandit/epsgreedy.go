package bandit

import (
	"fmt"
	"math/rand/v2"
)

// EpsilonGreedy explores a random arm with probability Epsilon and otherwise
// exploits the arm with the best average reward so far.
type EpsilonGreedy struct {
	armStats // embedded: brings counts, means, total and Update along

	Epsilon float64
	rng     *rand.Rand
}

// NewEpsilonGreedy returns a policy for nArms arms.
func NewEpsilonGreedy(nArms int, epsilon float64, rng *rand.Rand) (*EpsilonGreedy, error) {
	if nArms < 1 {
		return nil, ErrNoArms
	}
	if !(epsilon >= 0 && epsilon <= 1) {
		return nil, fmt.Errorf("%w: got %v", ErrInvalidEpsilon, epsilon)
	}
	return &EpsilonGreedy{armStats: newArmStats(nArms), Epsilon: epsilon, rng: rng}, nil
}

// Name identifies the policy in reports.
func (p *EpsilonGreedy) Name() string { return fmt.Sprintf("epsilon-greedy(%.2f)", p.Epsilon) }

// Select chooses the next arm to pull.
func (p *EpsilonGreedy) Select() int {
	if p.rng.Float64() < p.Epsilon {
		return p.rng.IntN(len(p.means))
	}
	return Argmax(p.means, p.rng)
}
