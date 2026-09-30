package bandit

import (
	"testing"
)

func TestArgmax(t *testing.T) {
	t.Run("floats", func(t *testing.T) {
		if got := Argmax([]float64{0.1, 0.7, 0.3}, nil); got != 1 {
			t.Errorf("got %d, want 1", got)
		}
	})
	t.Run("ints", func(t *testing.T) {
		if got := Argmax([]int{4, 2, 9, 9}, nil); got != 2 {
			t.Errorf("got %d, want 2 (lowest index wins ties)", got)
		}
	})
	t.Run("strings", func(t *testing.T) {
		if got := Argmax([]string{"a", "c", "b"}, nil); got != 1 {
			t.Errorf("got %d, want 1", got)
		}
	})
	t.Run("random ties are spread out", func(t *testing.T) {
		rng := NewRNG(1, StreamPolicy)
		counts := make([]int, 3)
		for range 3000 {
			counts[Argmax([]int{5, 5, 5}, rng)]++
		}
		for arm, n := range counts {
			if n < 800 || n > 1200 {
				t.Errorf("arm %d chosen %d/3000 times, want about 1000", arm, n)
			}
		}
	})
	t.Run("empty panics", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("expected a panic")
			}
		}()
		Argmax([]int{}, nil)
	})
}
