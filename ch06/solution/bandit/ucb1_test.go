package bandit

import (
	"slices"
	"testing"
)

func TestUCB1TriesEveryArmOnceFirst(t *testing.T) {
	p := newUCB(t, 4)
	var order []int
	for range 4 {
		arm := p.Select()
		order = append(order, arm)
		p.Update(arm, 0)
	}
	if !slices.Equal(order, []int{0, 1, 2, 3}) {
		t.Errorf("first four selections = %v, want [0 1 2 3]", order)
	}
}

func TestUCB1ExploitsWhenEvidenceIsStrong(t *testing.T) {
	p := newUCB(t, 2)
	p.counts, p.means, p.total = []int{50, 50}, []float64{0.9, 0.1}, 100
	if arm := p.Select(); arm != 0 {
		t.Errorf("Select() = %d, want 0: equal evidence, much better mean", arm)
	}
}

func TestUCB1ExploresARarelyTriedArm(t *testing.T) {
	p := newUCB(t, 2)
	// Arm 0 looks better (0.9 against 0.5), but arm 1 has been pulled twice
	// against a hundred, so its bonus is large: 0.5 + 2.15 beats 0.9 + 0.30.
	p.counts, p.means, p.total = []int{100, 2}, []float64{0.9, 0.5}, 102
	if arm := p.Select(); arm != 1 {
		t.Errorf("Select() = %d, want 1: the under-explored arm's bonus should win", arm)
	}
}

func TestUCB1TieGoesToTheFirstArm(t *testing.T) {
	p := newUCB(t, 3)
	p.counts, p.means, p.total = []int{10, 10, 10}, []float64{0.5, 0.5, 0.5}, 30
	if arm := p.Select(); arm != 0 {
		t.Errorf("Select() = %d, want 0 on a tie", arm)
	}
}

func TestUCB1IsDeterministic(t *testing.T) {
	a, b := newUCB(t, 3), newUCB(t, 3)
	rewards := []float64{1, 0, 0, 1, 1, 0, 1, 1, 1, 0, 0, 1}
	for i, r := range rewards {
		x, y := a.Select(), b.Select()
		if x != y {
			t.Fatalf("step %d: %d vs %d, UCB1 uses no randomness", i, x, y)
		}
		a.Update(x, r)
		b.Update(y, r)
	}
}

func newUCB(t *testing.T, arms int) *UCB1 {
	t.Helper()
	p, err := NewUCB1(arms)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestUCB1RejectsZeroArms(t *testing.T) {
	if _, err := NewUCB1(0); err == nil {
		t.Error("zero arms should be an error")
	}
}
