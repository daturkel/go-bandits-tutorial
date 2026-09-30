package bandit

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

func TestRegretAt(t *testing.T) {
	curve := []float64{0.5, 1.0, 1.0, 2.5}

	got, ok := RegretAt(curve, []int{1, 4, 2})
	if !ok {
		t.Fatal("ok = false, want true")
	}
	want := map[int]float64{1: 0.5, 4: 2.5, 2: 1.0}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("mark %d = %v, want %v", k, got[k], v)
		}
	}

	for _, bad := range [][]int{{0}, {5}, {2, -1}, {1, 99}} {
		if m, ok := RegretAt(curve, bad); ok || m != nil {
			t.Errorf("marks %v: got (%v, %v), want (nil, false)", bad, m, ok)
		}
	}

	if m, ok := RegretAt(curve, nil); !ok || len(m) != 0 {
		t.Errorf("no marks: got (%v, %v), want empty map and true", m, ok)
	}
}

func TestDecayedEpsilon(t *testing.T) {
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
			got := DecayedEpsilon(tc.t, tc.start, tc.floor, tc.halfLife)
			if math.Abs(got-tc.want) > 1e-12 {
				t.Errorf("DecayedEpsilon(%d, %v, %v, %v) = %v, want %v", tc.t, tc.start, tc.floor, tc.halfLife, got, tc.want)
			}
		})
	}
}
