package ex4

import (
	"slices"
	"testing"
)

func TestTally(t *testing.T) {
	tests := []struct {
		pulls []int
		arms  int
		want  []int
	}{
		{[]int{2, 0, 2, 2}, 3, []int{1, 0, 3}},
		{nil, 3, []int{0, 0, 0}},
		{[]int{0, 1, 1}, 2, []int{1, 2}},
		{[]int{5, -1, 1, 9}, 3, []int{0, 1, 0}},
		{[]int{0}, 0, []int{}},
	}
	for _, tc := range tests {
		if got := Tally(tc.pulls, tc.arms); !slices.Equal(got, tc.want) || got == nil {
			t.Errorf("Tally(%v, %d) = %v, want %v", tc.pulls, tc.arms, got, tc.want)
		}
	}
}

func TestTallyDoesNotChangeItsInput(t *testing.T) {
	in := []int{1, 1, 0}
	Tally(in, 2)
	if !slices.Equal(in, []int{1, 1, 0}) {
		t.Errorf("input changed to %v", in)
	}
}
