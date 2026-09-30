package ex1

import (
	"testing"
	"time"
)

func TestBackoff(t *testing.T) {
	const base, max = 100 * time.Millisecond, 5 * time.Second
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{5, 3200 * time.Millisecond},
		{6, 5 * time.Second}, // 6.4s would exceed the cap
		{10, 5 * time.Second},
		{62, 5 * time.Second},
		{63, 5 * time.Second},   // a plain shift by 63 overflows int64
		{1000, 5 * time.Second}, // and by 1000 wraps around
		{-3, 100 * time.Millisecond},
	}
	for _, tc := range tests {
		if got := Backoff(tc.attempt, base, max); got != tc.want {
			t.Errorf("Backoff(%d) = %v, want %v", tc.attempt, got, tc.want)
		}
	}
}

func TestBackoffEdgeCases(t *testing.T) {
	if got := Backoff(3, 0, time.Second); got != 0 {
		t.Errorf("zero base = %v, want 0", got)
	}
	if got := Backoff(3, time.Second, 0); got != 0 {
		t.Errorf("zero max = %v, want 0", got)
	}
	if got := Backoff(0, 10*time.Second, time.Second); got != time.Second {
		t.Errorf("base above max = %v, want max", got)
	}
}
