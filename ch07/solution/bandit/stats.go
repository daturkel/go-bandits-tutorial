// This file comes with the course, and later chapters may replace it with a
// new version. Keep your own code in other files.

package bandit

// armStats is the bookkeeping every policy needs: how often each arm was
// pulled and its running mean reward. Policies embed it to reuse the fields
// and methods.
type armStats struct {
	counts []int
	means  []float64
	total  int
}

func newArmStats(nArms int) armStats {
	return armStats{counts: make([]int, nArms), means: make([]float64, nArms)}
}

// Update folds an observed reward into the arm's running mean.
func (s *armStats) Update(arm int, reward float64) {
	s.counts[arm]++
	s.total++
	s.means[arm] += (reward - s.means[arm]) / float64(s.counts[arm])
}
