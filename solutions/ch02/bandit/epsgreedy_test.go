package bandit

import (
	"math"
	"testing"
)

// EpsilonGreedy now embeds armStats, so its counts and means are the shared
// ones. These tests check the behaviour that must not change with the move.

func newEG(arms int, eps float64) *EpsilonGreedy {
	return NewEpsilonGreedy(arms, eps, NewRNG(1, StreamPolicy))
}

func TestEpsilonGreedyLearnsThroughTheEmbeddedStats(t *testing.T) {
	p := newEG(3, 0.1)
	for _, r := range []float64{1, 0, 1, 1} {
		p.Update(1, r)
	}
	if p.counts[1] != 4 || p.total != 4 || math.Abs(p.means[1]-0.75) > 1e-12 {
		t.Errorf("count=%d total=%d mean=%v, want 4, 4 and 0.75", p.counts[1], p.total, p.means[1])
	}
}

func TestEpsilonGreedyNameShowsEpsilon(t *testing.T) {
	if got := newEG(2, 0.05).Name(); got != "epsilon-greedy(0.05)" {
		t.Errorf("Name() = %q", got)
	}
}

func TestEpsilonGreedyExploitsAndExplores(t *testing.T) {
	p := newEG(3, 0)
	p.means = []float64{0.2, 0.9, 0.4}
	for range 100 {
		if arm := p.Select(); arm != 1 {
			t.Fatalf("epsilon 0: Select() = %d, want 1", arm)
		}
	}
	q := newEG(4, 1)
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
