package ex1

import (
	"slices"
	"testing"
)

func TestDelta(t *testing.T) {
	older := []Totals{{10, 4}, {5, 5}, {0, 0}}
	newer := []Totals{{14, 6}, {5, 5}, {3, 1}}
	got, ok := Delta(newer, older)
	want := []Totals{{4, 2}, {0, 0}, {3, 1}}
	if !ok || !slices.Equal(got, want) {
		t.Errorf("Delta = %v, %v; want %v, true", got, ok, want)
	}
}

func TestDeltaOfIdenticalReadingsIsZero(t *testing.T) {
	x := []Totals{{7, 3}, {1, 0}}
	got, ok := Delta(x, x)
	if !ok || !slices.Equal(got, []Totals{{}, {}}) {
		t.Errorf("Delta(x, x) = %v, %v; want all zeros", got, ok)
	}
}

func TestDeltaRejectsBadReadings(t *testing.T) {
	if _, ok := Delta([]Totals{{1, 1}}, []Totals{{1, 1}, {2, 1}}); ok {
		t.Error("different lengths should not be ok")
	}
	if got, ok := Delta([]Totals{{3, 1}, {1, 0}}, []Totals{{2, 1}, {4, 2}}); ok || got != nil {
		t.Errorf("a pull count that went down should give (nil, false), got %v, %v", got, ok)
	}
}

func TestDeltaDoesNotModifyItsInputs(t *testing.T) {
	older := []Totals{{1, 1}}
	newer := []Totals{{3, 2}}
	Delta(newer, older)
	if older[0] != (Totals{1, 1}) || newer[0] != (Totals{3, 2}) {
		t.Error("inputs were modified")
	}
}
