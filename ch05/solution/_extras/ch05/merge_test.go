package extras

import (
	"slices"
	"testing"
	"time"
)

func feed(vals ...int) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)
		for _, v := range vals {
			ch <- v
		}
	}()
	return ch
}

// drain reads until ch is closed, failing the test if that takes too long.
func drain(t *testing.T, ch <-chan int) []int {
	t.Helper()
	if ch == nil {
		t.Fatal("Merge returned a nil channel")
	}
	var got []int
	timeout := time.After(2 * time.Second)
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				return got
			}
			got = append(got, v)
		case <-timeout:
			t.Fatalf("timed out after receiving %v; is the output closed when the inputs are?", got)
		}
	}
}

func TestMerge(t *testing.T) {
	got := drain(t, Merge(feed(1, 2, 3), feed(10, 20), feed(), feed(100)))
	slices.Sort(got)
	if want := []int{1, 2, 3, 10, 20, 100}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMergeNoInputs(t *testing.T) {
	if got := drain(t, Merge()); len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
}

func TestMergeSingleInputKeepsOrder(t *testing.T) {
	got := drain(t, Merge(feed(3, 1, 2)))
	if want := []int{3, 1, 2}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
