package ex5

// RunningMean keeps the average of the numbers added to it without storing
// them. Its zero value is an empty mean, ready to use: var m RunningMean.
type RunningMean struct {
	n    int
	mean float64
}

// Add includes x in the mean.
//
// Hint: after n numbers, newMean = oldMean + (x - oldMean) / n.
func (m *RunningMean) Add(x float64) {
	m.n++
	m.mean += (x - m.mean) / float64(m.n)
}

// Mean returns the average so far, or 0 if nothing was added.
func (m *RunningMean) Mean() float64 { return m.mean }

// N returns how many numbers were added.
func (m *RunningMean) N() int { return m.n }
