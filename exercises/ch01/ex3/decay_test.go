package ex3

import (
	"math"
	"testing"
)

func TestEpsilon(t *testing.T) {
	tests := []struct {
		name                         string
		t                            int
		start, floor, halfLife, want float64
	}{
		{"at the start", 0, 0.5, 0.01, 100, 0.5},
		{"one half-life", 100, 0.5, 0.01, 100, 0.25},
		{"two half-lives", 200, 0.5, 0.01, 100, 0.125},
		{"fractional half-life", 50, 1, 0, 100, math.Sqrt(0.5)},
		{"floor applies", 10_000, 0.5, 0.01, 100, 0.01},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Epsilon(tc.t, tc.start, tc.floor, tc.halfLife)
			if math.Abs(got-tc.want) > 1e-12 {
				t.Errorf("Epsilon(%d, %v, %v, %v) = %v, want %v", tc.t, tc.start, tc.floor, tc.halfLife, got, tc.want)
			}
		})
	}
}
