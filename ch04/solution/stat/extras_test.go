package stat

import (
	"math"
	"slices"
	"testing"
)

type result struct {
	Policy string
	Reward float64
	Pulls  int
}

func TestSumBy(t *testing.T) {
	results := []result{{"a", 1.5, 10}, {"b", 2.5, 20}, {"c", 3, 30}}

	if got := SumBy(results, func(r result) float64 { return r.Reward }); math.Abs(got-7) > 1e-12 {
		t.Errorf("sum of rewards = %v, want 7", got)
	}
	if got := SumBy(results, func(r result) int { return r.Pulls }); got != 60 {
		t.Errorf("sum of pulls = %v, want 60", got)
	}
	if got := SumBy([]string{"ab", "cde"}, func(s string) int { return len(s) }); got != 5 {
		t.Errorf("sum of lengths = %v, want 5", got)
	}
	if got := SumBy[result, float64](nil, func(r result) float64 { return r.Reward }); got != 0 {
		t.Errorf("empty sum = %v, want 0", got)
	}
}

func TestSumByNamedType(t *testing.T) {
	type Reward float64 // works because Number uses ~float64
	got := SumBy([]int{1, 2, 3}, func(i int) Reward { return Reward(i) / 2 })
	if got != 3 {
		t.Errorf("got %v, want 3", got)
	}
}

func TestMaxBy(t *testing.T) {
	results := []result{{"a", 1.5, 10}, {"b", 2.5, 20}, {"c", 2.5, 30}}

	best, ok := MaxBy(results, func(r result) float64 { return r.Reward })
	if !ok || best.Policy != "b" {
		t.Errorf("best by reward = %+v, %v; want policy b (first of the tie)", best, ok)
	}
	longest, ok := MaxBy([]string{"go", "python", "c"}, func(s string) int { return len(s) })
	if !ok || longest != "python" {
		t.Errorf("longest = %q, %v", longest, ok)
	}
	if got, ok := MaxBy([]result{}, func(r result) int { return r.Pulls }); ok || got != (result{}) {
		t.Errorf("empty input: got %+v, %v; want zero value and false", got, ok)
	}
}

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

func TestCounterZeroValue(t *testing.T) {
	var c Counter[string] // no constructor call
	if got := c.Get("x"); got != 0 {
		t.Errorf("Get on empty counter = %d, want 0", got)
	}
	if _, _, ok := c.MostCommon(); ok {
		t.Error("MostCommon on empty counter should report false")
	}
	c.Inc("x") // must not panic on a nil map
	c.Inc("x")
	c.Inc("y")
	if got := c.Get("x"); got != 2 {
		t.Errorf("Get(x) = %d, want 2", got)
	}
	key, n, ok := c.MostCommon()
	if !ok || key != "x" || n != 2 {
		t.Errorf("MostCommon = (%q, %d, %v), want (x, 2, true)", key, n, ok)
	}
}

func TestCounterWithIntKeys(t *testing.T) {
	var c Counter[int]
	for _, arm := range []int{2, 0, 2, 2, 1} {
		c.Inc(arm)
	}
	if key, n, ok := c.MostCommon(); !ok || key != 2 || n != 3 {
		t.Errorf("MostCommon = (%d, %d, %v), want (2, 3, true)", key, n, ok)
	}
}

type pair struct{ a, b int }

func TestCounterWithStructKeys(t *testing.T) {
	var c Counter[pair] // any comparable type works
	c.Inc(pair{1, 2})
	c.Inc(pair{1, 2})
	if got := c.Get(pair{1, 2}); got != 2 {
		t.Errorf("Get = %d, want 2", got)
	}
}
