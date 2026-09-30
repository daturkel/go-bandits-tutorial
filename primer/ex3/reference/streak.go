package ex3

// LongestStreak returns the length of the longest run of consecutive 1s in
// rewards. LongestStreak([]int{1, 1, 0, 1, 1, 1, 0}) is 3, and an empty slice
// gives 0.
func LongestStreak(rewards []int) int {
	best, current := 0, 0
	for _, r := range rewards {
		if r == 1 {
			current++
			best = max(best, current)
		} else {
			current = 0
		}
	}
	return best
}
