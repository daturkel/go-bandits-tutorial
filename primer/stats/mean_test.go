package stats

import "testing"

func TestMean(t *testing.T) {
	tests := []struct {
		name string
		in   []float64
		want float64
	}{
		{"empty", nil, 0},
		{"one", []float64{4}, 4},
		{"several", []float64{1, 2, 3, 6}, 3},
		{"negative", []float64{-1, 1}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Mean(tc.in); got != tc.want {
				t.Errorf("Mean(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
