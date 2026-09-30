package ex1

// Percent returns hits as a percentage of total: Percent(1, 4) is 25.
// If total is 0 there is nothing to divide by, so return 0.
func Percent(hits, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}
