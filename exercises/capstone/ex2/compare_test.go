package ex2

import "testing"

func TestCompare(t *testing.T) {
	tests := []struct {
		name string
		a, b []float64
		z    float64
		want Verdict
	}{
		{"a clearly lower", []float64{10, 11, 9, 10, 10}, []float64{20, 21, 19, 20, 20}, 2, AIsLower},
		{"b clearly lower", []float64{20, 21, 19, 20, 20}, []float64{10, 11, 9, 10, 10}, 2, BIsLower},
		{"overlapping noise", []float64{10, 14, 6, 12, 8}, []float64{11, 15, 7, 13, 9}, 2, Inconclusive},
		{"identical", []float64{5, 6, 7}, []float64{5, 6, 7}, 2, Inconclusive},
		{"one measurement", []float64{1}, []float64{100, 101, 99}, 2, Inconclusive},
		{"none", nil, []float64{1, 2}, 2, Inconclusive},
		{"no noise at all, different means", []float64{1, 1, 1}, []float64{2, 2, 2}, 2, AIsLower},
		{"no noise, same mean", []float64{3, 3}, []float64{3, 3}, 2, Inconclusive},
		{"a stricter z turns it inconclusive", []float64{10, 12, 8, 11, 9}, []float64{13, 15, 11, 14, 12}, 5, Inconclusive},
	}
	for _, tc := range tests {
		if got := Compare(tc.a, tc.b, tc.z); got != tc.want {
			t.Errorf("%s: Compare = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestCompareDoesNotModifyItsInputs(t *testing.T) {
	a, b := []float64{3, 1, 2}, []float64{9, 8, 7}
	Compare(a, b, 2)
	if a[0] != 3 || a[1] != 1 || b[0] != 9 {
		t.Errorf("inputs were modified: %v %v", a, b)
	}
}
