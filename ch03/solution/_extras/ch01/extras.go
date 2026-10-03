package extras

import (
	"math"
)

// EmpiricalRates returns wins[i] / (wins[i] + losses[i]) for every arm.
//
// An arm that has never been pulled has rate 0 (not NaN). If wins and losses
// have different lengths the input is inconsistent: return nil and false.
func EmpiricalRates(wins, losses []int) ([]float64, bool) {
	if len(wins) != len(losses) {
		return nil, false
	}
	rates := make([]float64, len(wins))
	for i := range wins {
		if n := wins[i] + losses[i]; n > 0 {
			rates[i] = float64(wins[i]) / float64(n)
		}
	}
	return rates, true
}

// RegretAt looks up the cumulative regret at the given step counts.
//
// curve[i] is the regret after step i+1, so mark 1 reads curve[0]. It returns
// a map from each mark to its regret. If any mark is below 1 or beyond the end
// of the curve, return nil and false.
func RegretAt(curve []float64, marks []int) (map[int]float64, bool) {
	out := make(map[int]float64, len(marks))
	for _, m := range marks {
		if m < 1 || m > len(curve) {
			return nil, false
		}
		out[m] = curve[m-1]
	}
	return out, true
}

// DecayedEpsilon returns an exploration rate that halves every halfLife steps:
//
//	start * 0.5^(t/halfLife)
//
// but never drops below floor. t is the number of steps taken so far.
func DecayedEpsilon(t int, start, floor, halfLife float64) float64 {
	return math.Max(floor, start*math.Pow(0.5, float64(t)/halfLife))
}
