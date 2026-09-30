package ex1

// Logged is one decision made by the policy that was running when the data
// was collected (the logging policy).
type Logged struct {
	Arm        int
	Reward     float64
	Propensity float64 // the probability the logging policy gave to Arm when it chose it
}

// IPS estimates the average reward a different policy would have earned, from
// data logged under another one, using inverse propensity scoring. target[a]
// is the probability the new policy gives to arm a. Each logged decision
// counts target[arm]/Propensity times its reward, so that a decision the
// logging policy rarely made, but the new policy likes, is weighted up.
//
//	estimate = (1/n) * sum over the log of  target[arm] / propensity * reward
//
// It returns ok == false if the log is empty, if any propensity is not in
// (0, 1], or if any arm is outside target: the estimate would be meaningless.
func IPS(log []Logged, target []float64) (estimate float64, ok bool) {
	if len(log) == 0 {
		return 0, false
	}
	var sum float64
	for _, d := range log {
		if !(d.Propensity > 0 && d.Propensity <= 1) || d.Arm < 0 || d.Arm >= len(target) {
			return 0, false
		}
		sum += target[d.Arm] / d.Propensity * d.Reward
	}
	return sum / float64(len(log)), true
}
