package bandit

import "math"

// UCB1 pulls the arm with the highest optimistic estimate: its mean plus a
// bonus that shrinks as the arm is pulled more often. It uses no randomness.
type UCB1 struct {
	armStats
}

// NewUCB1 returns a UCB1 policy for nArms arms.
func NewUCB1(nArms int) *UCB1 {
	return &UCB1{armStats: newArmStats(nArms)}
}

// Name identifies the policy in reports.
func (p *UCB1) Name() string { return "ucb1" }

// Select tries every arm once, then maximises mean + sqrt(2 ln t / n).
func (p *UCB1) Select() int {
	for arm, n := range p.counts {
		if n == 0 {
			return arm
		}
	}
	best, bestScore := 0, math.Inf(-1)
	for arm, n := range p.counts {
		score := p.means[arm] + math.Sqrt(2*math.Log(float64(p.total))/float64(n))
		if score > bestScore {
			best, bestScore = arm, score
		}
	}
	return best
}
