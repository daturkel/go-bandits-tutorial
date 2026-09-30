package ex3

import "testing"

func TestLongestStreak(t *testing.T) {
	tests := []struct {
		in   []int
		want int
	}{
		{nil, 0},
		{[]int{0, 0, 0}, 0},
		{[]int{1}, 1},
		{[]int{1, 1, 0, 1, 1, 1, 0}, 3},
		{[]int{1, 1, 1, 1}, 4},
		{[]int{0, 1, 0, 1, 0}, 1},
		{[]int{1, 1, 1, 0, 1, 1}, 3},
	}
	for _, tc := range tests {
		if got := LongestStreak(tc.in); got != tc.want {
			t.Errorf("LongestStreak(%v) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
