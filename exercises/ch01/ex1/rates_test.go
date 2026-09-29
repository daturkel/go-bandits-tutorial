package ex1

import (
	"math"
	"testing"
)

func TestEmpiricalRates(t *testing.T) {
	tests := []struct {
		name         string
		wins, losses []int
		want         []float64
		wantOK       bool
	}{
		{"basic", []int{1, 3}, []int{3, 1}, []float64{0.25, 0.75}, true},
		{"unpulled arm is zero", []int{0, 2}, []int{0, 2}, []float64{0, 0.5}, true},
		{"empty", []int{}, []int{}, []float64{}, true},
		{"length mismatch", []int{1}, []int{1, 2}, nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := EmpiricalRates(tc.wins, tc.losses)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				if got != nil {
					t.Fatalf("got %v, want nil on failure", got)
				}
				return
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if math.IsNaN(got[i]) || math.Abs(got[i]-tc.want[i]) > 1e-12 {
					t.Errorf("rate[%d] = %v, want %v", i, got[i], tc.want[i])
				}
			}
		})
	}
}
