package ex2

import (
	"math"
	"slices"
	"testing"
)

func TestQuantile(t *testing.T) {
	tests := []struct {
		name string
		xs   []float64
		q    float64
		want float64
	}{
		{"median even", []float64{1, 2, 3, 4}, 0.5, 2.5},
		{"median odd", []float64{5, 1, 3}, 0.5, 3},
		{"min", []float64{4, 2, 9}, 0, 2},
		{"max", []float64{4, 2, 9}, 1, 9},
		{"quarter", []float64{0, 10, 20, 30, 40}, 0.25, 10},
		{"interpolated", []float64{0, 10}, 0.3, 3},
		{"single", []float64{7}, 0.9, 7},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Quantile(tc.xs, tc.q); math.Abs(got-tc.want) > 1e-12 {
				t.Errorf("Quantile(%v, %v) = %v, want %v", tc.xs, tc.q, got, tc.want)
			}
		})
	}
}

func TestQuantileEdgeCases(t *testing.T) {
	for _, q := range []float64{-0.1, 1.1, math.NaN()} {
		if got := Quantile([]float64{1, 2}, q); !math.IsNaN(got) {
			t.Errorf("q=%v: got %v, want NaN", q, got)
		}
	}
	if got := Quantile(nil, 0.5); !math.IsNaN(got) {
		t.Errorf("empty input: got %v, want NaN", got)
	}
}

func TestQuantileDoesNotSortTheInput(t *testing.T) {
	xs := []float64{3, 1, 2}
	Quantile(xs, 0.5)
	if !slices.Equal(xs, []float64{3, 1, 2}) {
		t.Errorf("input was modified: %v", xs)
	}
}
