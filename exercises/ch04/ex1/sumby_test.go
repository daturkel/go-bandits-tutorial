package ex1

import (
	"math"
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
