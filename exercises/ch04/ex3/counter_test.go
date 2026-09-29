package ex3

import "testing"

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
