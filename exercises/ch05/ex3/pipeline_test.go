package ex3

import (
	"slices"
	"testing"
	"time"
)

func collect(t *testing.T, ch <-chan int) []int {
	t.Helper()
	if ch == nil {
		t.Fatal("got a nil channel")
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
			t.Fatalf("timed out after %v; is the channel closed at the end?", got)
		}
	}
}

func TestGenerate(t *testing.T) {
	if got := collect(t, Generate(4, 5, 6)); !slices.Equal(got, []int{4, 5, 6}) {
		t.Errorf("got %v", got)
	}
	if got := collect(t, Generate()); len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
}

func TestSquare(t *testing.T) {
	in := make(chan int)
	out := Square(in)
	go func() {
		defer close(in)
		for _, v := range []int{1, -2, 3} {
			in <- v
		}
	}()
	if got := collect(t, out); !slices.Equal(got, []int{1, 4, 9}) {
		t.Errorf("got %v", got)
	}
}

func TestPipeline(t *testing.T) {
	got := collect(t, Square(Square(Generate(1, 2, 3))))
	if want := []int{1, 16, 81}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
