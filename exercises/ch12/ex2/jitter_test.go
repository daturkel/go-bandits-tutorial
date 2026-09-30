package ex2

import (
	"math/rand/v2"
	"testing"
	"time"
)

func TestJitterStaysInRange(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	const iv = 10 * time.Second
	lo, hi := iv, iv
	for range 1000 {
		d := Jitter(iv, 0.2, r)
		if d < 8*time.Second || d > 12*time.Second {
			t.Fatalf("Jitter = %v, want within [8s, 12s]", d)
		}
		lo, hi = min(lo, d), max(hi, d)
	}
	if lo > 8500*time.Millisecond || hi < 11500*time.Millisecond {
		t.Errorf("1000 draws only covered [%v, %v]; the jitter is not using its whole range", lo, hi)
	}
}

func TestJitterZeroFractionIsExact(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 10 {
		if d := Jitter(3*time.Second, 0, r); d != 3*time.Second {
			t.Fatalf("Jitter with frac 0 = %v, want 3s", d)
		}
	}
}

func TestJitterClampsFraction(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 1000 {
		if d := Jitter(time.Second, 5, r); d < 0 || d > 2*time.Second {
			t.Fatalf("Jitter with frac 5 = %v, want within [0, 2s]", d)
		}
		if d := Jitter(time.Second, -1, r); d != time.Second {
			t.Fatalf("Jitter with negative frac = %v, want exactly 1s", d)
		}
	}
}

func TestJitterIsDeterministicForASeed(t *testing.T) {
	a, b := rand.New(rand.NewPCG(7, 7)), rand.New(rand.NewPCG(7, 7))
	for range 20 {
		if x, y := Jitter(time.Minute, 0.5, a), Jitter(time.Minute, 0.5, b); x != y {
			t.Fatalf("same seed gave %v and %v", x, y)
		}
	}
}
