package bandit

import "testing"

func TestNewRNGStreams(t *testing.T) {
	a, b := NewRNG(1, StreamEnv), NewRNG(1, StreamEnv)
	if a.Uint64() != b.Uint64() {
		t.Error("same seed and stream must give the same numbers")
	}
	c, d := NewRNG(1, StreamEnv), NewRNG(1, StreamPolicy)
	if c.Uint64() == d.Uint64() {
		t.Error("different streams should give different numbers")
	}
}
