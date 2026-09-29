package ex1

import (
	"errors"
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
		if !errors.Is(err, ErrOutOfRange) {
			t.Errorf("%q: error %v does not wrap ErrOutOfRange", in, err)
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
