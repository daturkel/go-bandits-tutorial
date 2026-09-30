package stat

import (
	"math"
	"slices"
	"testing"
)

type reward float64 // a named type: only ~float64 admits it

func TestMean(t *testing.T) {
	if got := Mean([]int{1, 2, 3, 4}); got != 2.5 {
		t.Errorf("Mean ints = %v", got)
	}
	if got := Mean([]reward{0.5, 1}); got != 0.75 {
		t.Errorf("Mean rewards = %v", got)
	}
	if !math.IsNaN(Mean([]float64{})) {
		t.Error("Mean of empty slice should be NaN")
	}
}

func TestStdErr(t *testing.T) {
	// sample sd of 2,4,4,4,5,5,7,9 is sqrt(32/7); the standard error divides by sqrt(8).
	want := math.Sqrt(32.0/7) / math.Sqrt(8)
	if got := StdErr([]int{2, 4, 4, 4, 5, 5, 7, 9}); math.Abs(got-want) > 1e-12 {
		t.Errorf("StdErr = %v, want %v", got, want)
	}
	if StdErr([]float64{3}) != 0 {
		t.Error("StdErr of one sample should be 0")
	}
}

func TestSample(t *testing.T) {
	xs := []string{"a", "b", "c", "d", "e", "f", "g"}
	tests := []struct {
		n    int
		want []string
	}{
		{n: 3, want: []string{"a", "d", "g"}},
		{n: 2, want: []string{"a", "g"}},
		{n: 1, want: []string{"g"}},
		{n: 50, want: xs},
	}
	for _, tc := range tests {
		if got := Sample(xs, tc.n); !slices.Equal(got, tc.want) {
			t.Errorf("Sample(n=%d) = %v, want %v", tc.n, got, tc.want)
		}
	}
}

func TestQuantile(t *testing.T) {
	xs := []float64{4, 1, 3, 2}
	tests := []struct {
		q, want float64
	}{{0, 1}, {1, 4}, {0.5, 2.5}, {0.25, 1.75}}
	for _, tc := range tests {
		if got := Quantile(xs, tc.q); math.Abs(got-tc.want) > 1e-12 {
			t.Errorf("Quantile(%v) = %v, want %v", tc.q, got, tc.want)
		}
	}
	if !slices.Equal(xs, []float64{4, 1, 3, 2}) {
		t.Errorf("Quantile sorted its input: %v", xs)
	}
	for _, q := range []float64{-0.1, 1.1, math.NaN()} {
		if !math.IsNaN(Quantile(xs, q)) {
			t.Errorf("Quantile(%v) should be NaN", q)
		}
	}
	if !math.IsNaN(Quantile(nil, 0.5)) {
		t.Error("Quantile of nothing should be NaN")
	}
}
