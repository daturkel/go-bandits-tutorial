package ex3

// Tally counts pulls per arm and in total.
//
// It has a bug: the test below fails even though every method looks
// reasonable. Find it and fix it. Do not change the tests.
type Tally struct {
	counts []int
	total  int
}

// NewTally returns a Tally for nArms arms.
func NewTally(nArms int) Tally {
	return Tally{counts: make([]int, nArms)}
}

// Add records one pull of arm.
func (t Tally) Add(arm int) {
	t.counts[arm]++
	t.total++
}

// Total returns the number of pulls recorded.
func (t Tally) Total() int { return t.total }

// Count returns the number of pulls of arm.
func (t Tally) Count(arm int) int { return t.counts[arm] }
