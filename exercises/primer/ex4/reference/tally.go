package ex4

// Tally counts how many times each arm appears in pulls. The result has one
// entry per arm: Tally([]int{2, 0, 2, 2}, 3) is [1 0 3]. Entries in pulls
// that are not a valid arm (negative, or arms or more) are ignored.
func Tally(pulls []int, arms int) []int {
	counts := make([]int, arms)
	for _, arm := range pulls {
		if arm >= 0 && arm < arms {
			counts[arm]++
		}
	}
	return counts
}
