package bandit

import (
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestParseProbsOK(t *testing.T) {
	got, err := ParseProbs("0.2,0.5,1")
	if err != nil {
		t.Fatal(err)
	}
	if want := []float64{0.2, 0.5, 1}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseProbsRange(t *testing.T) {
	for _, in := range []string{"0.2,1.5", "-0.1", "0.5,0.5,2"} {
		_, err := ParseProbs(in)
		if !errors.Is(err, ErrInvalidProbability) {
			t.Errorf("%q: error %v does not wrap ErrInvalidProbability", in, err)
		}
	}
}

func TestParseProbsSyntax(t *testing.T) {
	_, err := ParseProbs("0.2,abc,0.4")
	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		t.Fatalf("error %v does not wrap a *strconv.NumError", err)
	}
	if numErr.Num != "abc" {
		t.Errorf("NumError.Num = %q, want %q", numErr.Num, "abc")
	}
	if !strings.Contains(err.Error(), "entry 1") {
		t.Errorf("error %q should name the bad entry as %q", err, "entry 1")
	}
}

func TestParseProbsNamesTheEntry(t *testing.T) {
	_, err := ParseProbs("0.1,0.2,7")
	if err == nil || !strings.Contains(err.Error(), "entry 2") {
		t.Errorf("error %v should mention %q", err, "entry 2")
	}
}

func TestParseProbsEmpty(t *testing.T) {
	if _, err := ParseProbs(""); err == nil {
		t.Error("empty input should be an error")
	}
}

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
