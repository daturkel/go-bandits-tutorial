package ex1

import "testing"

// This line only compiles if *RoundRobin has every method of Policy.
var _ Policy = (*RoundRobin)(nil)

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
