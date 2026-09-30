package ex1

import (
	"slices"
	"sync/atomic"
	"testing"
	"time"
)

func TestParallelMapOrder(t *testing.T) {
	got := ParallelMap([]int{1, 2, 3, 4, 5}, func(x int) string {
		time.Sleep(time.Duration(6-x) * time.Millisecond) // finish in reverse order
		return string(rune('a' + x))
	})
	want := []string{"b", "c", "d", "e", "f"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v (results must follow input order)", got, want)
	}
}

func TestParallelMapEmpty(t *testing.T) {
	got := ParallelMap([]int{}, func(x int) int { return x })
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

// Every call waits until all n calls are in flight together. A sequential
// implementation can never satisfy that.
func TestParallelMapRunsConcurrently(t *testing.T) {
	const n = 8
	var arrived atomic.Int32
	release := make(chan struct{})
	f := func(x int) int {
		if arrived.Add(1) == n {
			close(release)
		}
		select {
		case <-release:
		case <-time.After(2 * time.Second):
			t.Error("calls did not overlap: ParallelMap is not concurrent")
		}
		return x * 2
	}
	xs := make([]int, n)
	for i := range xs {
		xs[i] = i
	}
	got := ParallelMap(xs, f)
	if len(got) != n || got[n-1] != 2*(n-1) {
		t.Errorf("got %v", got)
	}
}
