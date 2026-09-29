package ex1

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
