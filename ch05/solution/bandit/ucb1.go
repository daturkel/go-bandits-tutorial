package bandit

import "math"

// UCB1 pulls the arm with the highest optimistic estimate: its mean plus a
// bonus that shrinks as the arm is pulled more often. It uses no randomness.
type UCB1 struct {
	armStats
	scores []float64 // scratch space reused by every Select
}

// NewUCB1 returns a UCB1 policy for nArms arms.
func NewUCB1(nArms int) (*UCB1, error) {
	if nArms < 1 {
		return nil, ErrNoArms
	}
	return &UCB1{armStats: newArmStats(nArms), scores: make([]float64, nArms)}, nil
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
	for arm, n := range p.counts {
		p.scores[arm] = p.means[arm] + math.Sqrt(2*math.Log(float64(p.total))/float64(n))
	}
	return Argmax(p.scores, nil)
}
