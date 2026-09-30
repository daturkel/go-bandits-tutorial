package ex1

// Percent returns hits as a percentage of total, on a scale of 0 to 100:
// Percent(1, 4) is 25, not 0.25. The caller guarantees total is positive.
func Percent(hits, total int) float64 {
	return float64(hits) / float64(total) * 100
}
