package ex3

import "math"

// Softmax turns scores into a probability distribution: arm a gets
//
//	exp(score[a]/temp) / sum over b of exp(score[b]/temp)
//
// A low temperature concentrates the probability on the best-scoring arm; a
// high one spreads it out. Policies that log the probability they gave to the
// arm they chose make their data usable for off-policy evaluation.
//
// It must not overflow: scores can be large (a total reward of 5000), and
// exp(5000) is infinity. Subtract the maximum score before exponentiating;
// the result is the same and every exponent is at most 0.
//
// It returns nil if scores is empty or temp is not positive.
func Softmax(scores []float64, temp float64) []float64 {
	if len(scores) == 0 || !(temp > 0) {
		return nil
	}
	top := scores[0]
	for _, s := range scores {
		top = max(top, s)
	}
	p := make([]float64, len(scores))
	var total float64
	for i, s := range scores {
		p[i] = math.Exp((s - top) / temp)
		total += p[i]
	}
	for i := range p {
		p[i] /= total
	}
	return p
}
