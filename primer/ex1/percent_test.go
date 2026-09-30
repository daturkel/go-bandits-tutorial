package ex1

import (
	"math"
	"testing"
)

func TestPercent(t *testing.T) {
	tests := []struct {
		hits, total int
		want        float64
	}{
		{1, 4, 25},
		{0, 10, 0},
		{10, 10, 100},
		{1, 3, 100.0 / 3},
	}
	for _, tc := range tests {
		if got := Percent(tc.hits, tc.total); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("Percent(%d, %d) = %v, want %v", tc.hits, tc.total, got, tc.want)
		}
	}
}
