package ex3

import (
	"math"
	"testing"
)

func TestNormalize(t *testing.T) {
	dst := make([]float64, 0, 8)
	got := Normalize(dst, []float64{1, 1, 2})
	want := []float64{0.25, 0.25, 0.5}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-12 {
			t.Errorf("got[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestNormalizeZeroSum(t *testing.T) {
	got := Normalize(make([]float64, 0, 4), []float64{0, 0, 0})
	if len(got) != 3 || got[0] != 0 || got[1] != 0 || got[2] != 0 {
		t.Errorf("got %v, want three zeros", got)
	}
}

func TestNormalizeReusesDst(t *testing.T) {
	// A stale value in dst's backing array must not leak into the result.
	backing := []float64{9, 9, 9, 9}
	got := Normalize(backing[:0], []float64{0, 0})
	if len(got) != 2 || got[0] != 0 || got[1] != 0 {
		t.Errorf("got %v, want [0 0]", got)
	}
}

func TestNormalizeDoesNotAllocate(t *testing.T) {
	dst := make([]float64, 0, 16)
	src := []float64{1, 2, 3, 4}
	allocs := testing.AllocsPerRun(100, func() {
		dst = Normalize(dst[:0], src)
	})
	if allocs != 0 {
		t.Errorf("Normalize allocated %v times per call, want 0", allocs)
	}
}

func BenchmarkNormalize(b *testing.B) {
	dst := make([]float64, 0, 16)
	src := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	b.ReportAllocs()
	for b.Loop() {
		dst = Normalize(dst[:0], src)
	}
}
