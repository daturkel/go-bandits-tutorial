package ex2

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
