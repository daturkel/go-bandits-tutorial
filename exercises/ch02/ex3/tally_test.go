package ex3

import "testing"

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
