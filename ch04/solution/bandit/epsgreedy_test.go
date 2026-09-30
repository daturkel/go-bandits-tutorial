package bandit

import (
	"math"
	"testing"
)

func newEG(t *testing.T, arms int, eps float64) *EpsilonGreedy {
	t.Helper()
	p, err := NewEpsilonGreedy(arms, eps, NewRNG(1, StreamPolicy))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEpsilonGreedyLearnsThroughTheEmbeddedStats(t *testing.T) {
	p := newEG(t, 3, 0.1)
	for _, r := range []float64{1, 0, 1, 1} {
		p.Update(1, r)
	}
	if p.counts[1] != 4 || p.total != 4 || math.Abs(p.means[1]-0.75) > 1e-12 {
		t.Errorf("count=%d total=%d mean=%v, want 4, 4 and 0.75", p.counts[1], p.total, p.means[1])
	}
}

func TestEpsilonGreedyNameShowsEpsilon(t *testing.T) {
	if got := newEG(t, 2, 0.05).Name(); got != "epsilon-greedy(0.05)" {
		t.Errorf("Name() = %q", got)
	}
}

func TestEpsilonGreedyExploitsAndExplores(t *testing.T) {
	p := newEG(t, 3, 0)
	p.means = []float64{0.2, 0.9, 0.4}
	for range 100 {
		if arm := p.Select(); arm != 1 {
			t.Fatalf("epsilon 0: Select() = %d, want 1", arm)
		}
	}
	q := newEG(t, 4, 1)
	q.means = []float64{0.9, 0, 0, 0}
	seen := make([]int, 4)
	for range 4000 {
		seen[q.Select()]++
	}
	for arm, c := range seen {
		if c < 700 || c > 1300 {
			t.Errorf("epsilon 1: arm %d chosen %d of 4000 times (%v)", arm, c, seen)
		}
	}
}

// Tie-breaking must be random among equal arms, otherwise the first arm is
// favoured before any data has arrived.
func TestGreedyBreaksTiesAcrossAllArms(t *testing.T) {
	p := newEG(t, 4, 0)
	seen := map[int]bool{}
	for range 200 {
		seen[p.Select()] = true
	}
	if len(seen) != 4 {
		t.Errorf("greedy tie-break only ever chose arms %v", seen)
	}
}

func TestEpsilonGreedyRejectsBadArguments(t *testing.T) {
	if _, err := NewEpsilonGreedy(0, 0.1, NewRNG(1, StreamPolicy)); err == nil {
		t.Error("zero arms should be an error")
	}
	for _, eps := range []float64{-0.1, 1.5, math.NaN()} {
		if _, err := NewEpsilonGreedy(3, eps, NewRNG(1, StreamPolicy)); err == nil {
			t.Errorf("epsilon %v should be an error", eps)
		}
	}
}
