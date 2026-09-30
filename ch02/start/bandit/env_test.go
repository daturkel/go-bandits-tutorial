package bandit

import (
	"math"
	"testing"
)

func TestPullReturnsOnlyZeroOrOne(t *testing.T) {
	env := NewEnv([]float64{0.5}, NewRNG(1, StreamEnv))
	for range 1000 {
		if r := env.Pull(0); r != 0 && r != 1 {
			t.Fatalf("Pull returned %v, want 0 or 1", r)
		}
	}
}

func TestPullMatchesTheArmProbability(t *testing.T) {
	env := NewEnv([]float64{0, 0.3, 1}, NewRNG(1, StreamEnv))
	const n = 20000
	for arm, want := range []float64{0, 0.3, 1} {
		wins := 0.0
		for range n {
			wins += env.Pull(arm)
		}
		if got := wins / n; math.Abs(got-want) > 0.02 {
			t.Errorf("arm %d paid %.3f of the time, want about %.2f", arm, got, want)
		}
	}
}

func TestPullIsDeterministicForASeed(t *testing.T) {
	a := NewEnv([]float64{0.5, 0.5}, NewRNG(9, StreamEnv))
	b := NewEnv([]float64{0.5, 0.5}, NewRNG(9, StreamEnv))
	for i := range 200 {
		if x, y := a.Pull(i%2), b.Pull(i%2); x != y {
			t.Fatalf("same seed, different rewards at pull %d", i)
		}
	}
}

func TestBest(t *testing.T) {
	tests := []struct {
		name    string
		probs   []float64
		wantArm int
		wantP   float64
	}{
		{"one arm", []float64{0.3}, 0, 0.3},
		{"best last", []float64{0.2, 0.5, 0.8}, 2, 0.8},
		{"best first", []float64{0.9, 0.1}, 0, 0.9},
		{"best in the middle", []float64{0.1, 0.7, 0.4}, 1, 0.7},
		{"tie goes to the first", []float64{0.5, 0.5}, 0, 0.5},
		{"all zero", []float64{0, 0, 0}, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			arm, p := NewEnv(tc.probs, NewRNG(1, StreamEnv)).Best()
			if arm != tc.wantArm || p != tc.wantP {
				t.Errorf("Best() = (%d, %v), want (%d, %v)", arm, p, tc.wantArm, tc.wantP)
			}
		})
	}
}

func TestNewEnvCopiesItsInput(t *testing.T) {
	probs := []float64{0.1, 0.9}
	env := NewEnv(probs, NewRNG(1, StreamEnv))
	probs[0] = 0.99
	if env.Prob(0) != 0.1 {
		t.Errorf("changing the caller's slice changed the environment: Prob(0) = %v", env.Prob(0))
	}
}
