package bandit

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
	// TASK 2 (you will need to import "math"):
	// 1. If some arm has never been pulled (p.counts[arm] == 0), return it.
	// 2. Otherwise return the arm with the largest score, where
	//      score = p.means[arm] + math.Sqrt(2*math.Log(float64(p.total))/float64(p.counts[arm]))
	//    p.total is the number of pulls so far. If scores tie, keep the
	//    first arm.
	return 0
}
