package bandit

import (
	"slices"
	"testing"
)

var _ Policy = (*RoundRobin)(nil) // fails to compile unless RoundRobin has every method of Policy

func TestRoundRobinCycles(t *testing.T) {
	p := NewRoundRobin(3)
	if p.Name() != "round-robin" {
		t.Errorf("Name() = %q", p.Name())
	}
	want := []int{0, 1, 2, 0, 1, 2, 0}
	for i, w := range want {
		got := p.Select()
		p.Update(got, 1) // must not disturb the cycle
		if got != w {
			t.Fatalf("pull %d: got arm %d, want %d", i, got, w)
		}
	}
}

func TestRoundRobinSingleArm(t *testing.T) {
	p := NewRoundRobin(1)
	for range 5 {
		if got := p.Select(); got != 0 {
			t.Fatalf("got arm %d, want 0", got)
		}
	}
}

func TestRoundRobinInstancesAreIndependent(t *testing.T) {
	a, b := NewRoundRobin(2), NewRoundRobin(2)
	a.Select()
	if got := b.Select(); got != 0 {
		t.Errorf("second policy started at arm %d, want 0", got)
	}
}

type fake struct{ selects, updates int }

func (f *fake) Name() string        { return "fake" }
func (f *fake) Select() int         { f.selects++; return 2 }
func (f *fake) Update(int, float64) { f.updates++ }

func TestLoggedRecordsAndDelegates(t *testing.T) {
	inner := &fake{}
	var p Policy = NewLogged(inner) // *Logged must satisfy Policy

	if p.Name() != "fake" {
		t.Errorf("Name() = %q, want the wrapped policy's name", p.Name())
	}
	if got := p.Select(); got != 2 {
		t.Errorf("Select() = %d, want 2 from the wrapped policy", got)
	}
	p.Update(1, 1)
	p.Update(0, 0.5)

	if inner.selects != 1 || inner.updates != 2 {
		t.Errorf("wrapped policy saw %d selects and %d updates, want 1 and 2", inner.selects, inner.updates)
	}
	want := []string{"select 2", "update 1 1", "update 0 0.5"}
	if got := p.(*Logged).Log; !slices.Equal(got, want) {
		t.Errorf("Log = %q, want %q", got, want)
	}
}

func TestTally(t *testing.T) {
	tally := NewTally(3)
	tally.Add(0)
	tally.Add(2)
	tally.Add(2)

	if got := tally.Count(2); got != 2 {
		t.Errorf("Count(2) = %d, want 2", got)
	}
	if got := tally.Total(); got != 3 {
		t.Errorf("Total() = %d, want 3", got)
	}
}

// Counter is satisfied by whatever NewTally returns, however you fix it.
type Counter interface {
	Add(arm int)
	Total() int
}

func TestTallyThroughInterface(t *testing.T) {
	var c Counter = NewTally(2)
	c.Add(1)
	c.Add(1)
	if got := c.Total(); got != 2 {
		t.Errorf("Total() through the interface = %d, want 2", got)
	}
}
