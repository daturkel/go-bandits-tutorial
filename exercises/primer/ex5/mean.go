package ex5

// RunningMean keeps the average of the numbers added to it without storing
// them. Its zero value is an empty mean, ready to use: var m RunningMean.
type RunningMean struct {
	// TODO: fields
}

// Add includes x in the mean.
//
// Hint: after n numbers, newMean = oldMean + (x - oldMean) / n.
func (m *RunningMean) Add(x float64) {
	// TODO
}

// Mean returns the average so far, or 0 if nothing was added.
func (m *RunningMean) Mean() float64 {
	// TODO
	return 0
}

// N returns how many numbers were added.
func (m *RunningMean) N() int {
	// TODO
	return 0
}
