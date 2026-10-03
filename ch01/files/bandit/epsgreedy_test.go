package bandit

import (
	"math"
	"testing"
)

func newPolicy(arms int, eps float64) *EpsilonGreedy {
	return NewEpsilonGreedy(arms, eps, NewRNG(1, StreamPolicy))
}

func TestUpdateKeepsARunningMean(t *testing.T) {
	p := newPolicy(3, 0.1)
	for _, r := range []float64{1, 0, 1, 1} {
		p.Update(1, r)
	}
	if p.counts[1] != 4 || math.Abs(p.values[1]-0.75) > 1e-12 {
		t.Errorf("after rewards 1,0,1,1: count=%d mean=%v, want 4 and 0.75", p.counts[1], p.values[1])
	}
	if p.counts[0] != 0 || p.counts[2] != 0 || p.values[0] != 0 || p.values[2] != 0 {
		t.Errorf("other arms changed: counts=%v values=%v", p.counts, p.values)
	}
}

func TestGreedyPicksTheLargestMean(t *testing.T) {
	p := newPolicy(4, 0)
	p.values = []float64{0.1, 0.4, 0.9, 0.3}
	for range 50 {
		if arm := p.greedy(); arm != 2 {
			t.Fatalf("greedy() = %d, want 2", arm)
		}
	}
}

func TestGreedyBreaksTiesAtRandom(t *testing.T) {
	p := newPolicy(4, 0) // every mean is 0: all four arms tie
	seen := make([]int, 4)
	const n = 4000
	for range n {
		seen[p.greedy()]++
	}
	for arm, c := range seen {
		if c < n/4-300 || c > n/4+300 {
			t.Errorf("arm %d was picked %d times out of %d; ties should be broken evenly (%v)", arm, c, n, seen)
		}
	}
}

func TestGreedyTieAmongLeadersOnly(t *testing.T) {
	p := newPolicy(4, 0)
	p.values = []float64{0.5, 0.9, 0.9, 0.1}
	seen := make([]int, 4)
	for range 2000 {
		seen[p.greedy()]++
	}
	if seen[0] != 0 || seen[3] != 0 || seen[1] < 700 || seen[2] < 700 {
		t.Errorf("picks = %v; want only arms 1 and 2, about half each", seen)
	}
}

func TestSelectWithEpsilonZeroAlwaysExploits(t *testing.T) {
	p := newPolicy(3, 0)
	p.values = []float64{0.2, 0.9, 0.4}
	for range 200 {
		if arm := p.Select(); arm != 1 {
			t.Fatalf("Select() = %d, want 1 every time", arm)
		}
	}
}

func TestSelectWithEpsilonOneAlwaysExplores(t *testing.T) {
	p := newPolicy(4, 1)
	p.values = []float64{0.9, 0, 0, 0}
	seen := make([]int, 4)
	const n = 8000
	for range n {
		seen[p.Select()]++
	}
	for arm, c := range seen {
		if c < n/4-300 || c > n/4+300 {
			t.Errorf("arm %d chosen %d of %d times; exploring should be uniform (%v)", arm, c, n, seen)
		}
	}
}

func TestSelectMixesExploringAndExploiting(t *testing.T) {
	p := newPolicy(2, 0.5)
	p.values = []float64{0.9, 0.1}
	const n = 20000
	best := 0
	for range n {
		if p.Select() == 0 {
			best++
		}
	}
	// Half the time it exploits arm 0; half the time it picks an arm at
	// random, and that is arm 0 half the time: 0.5 + 0.25.
	if got := float64(best) / n; math.Abs(got-0.75) > 0.02 {
		t.Errorf("arm 0 chosen %.3f of the time, want about 0.75", got)
	}
}
