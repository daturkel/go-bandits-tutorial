package ex5

import (
	"math"
	"testing"
)

func TestRunningMean(t *testing.T) {
	var m RunningMean // no constructor needed: the zero value works
	if m.N() != 0 || m.Mean() != 0 {
		t.Fatalf("empty: N=%d Mean=%v, want 0 and 0", m.N(), m.Mean())
	}
	for i, x := range []float64{2, 4, 9, 1} {
		m.Add(x)
		if m.N() != i+1 {
			t.Errorf("N = %d after %d numbers", m.N(), i+1)
		}
	}
	if math.Abs(m.Mean()-4) > 1e-12 {
		t.Errorf("Mean = %v, want 4", m.Mean())
	}
}

func TestSeparateMeansAreIndependent(t *testing.T) {
	var a, b RunningMean
	a.Add(10)
	b.Add(2)
	if a.Mean() != 10 || b.Mean() != 2 {
		t.Errorf("a=%v b=%v, want 10 and 2", a.Mean(), b.Mean())
	}
}

func TestAddingManyNumbersStaysAccurate(t *testing.T) {
	var m RunningMean
	for i := range 100_000 {
		m.Add(float64(i % 10))
	}
	if math.Abs(m.Mean()-4.5) > 1e-9 {
		t.Errorf("Mean = %v, want 4.5", m.Mean())
	}
}
