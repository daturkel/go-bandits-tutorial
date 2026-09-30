package bandit

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestNewEnvValidation(t *testing.T) {
	tests := []struct {
		name    string
		probs   []float64
		wantErr error // matched with errors.Is; nil means success
		badArms []int // arms expected to be reported via ArmError
	}{
		{name: "valid", probs: []float64{0, 0.5, 1}},
		{name: "empty", probs: nil, wantErr: ErrNoArms},
		{name: "above one", probs: []float64{0.5, 1.2}, wantErr: ErrInvalidProbability, badArms: []int{1}},
		{name: "negative", probs: []float64{-0.1, 0.5}, wantErr: ErrInvalidProbability, badArms: []int{0}},
		{name: "nan", probs: []float64{math.NaN()}, wantErr: ErrInvalidProbability, badArms: []int{0}},
		{name: "reports all", probs: []float64{2, 0.5, -1}, wantErr: ErrInvalidProbability, badArms: []int{0, 2}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewEnv(tc.probs, NewRNG(1, StreamEnv))
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want one wrapping %v", err, tc.wantErr)
			}
			if len(tc.badArms) == 0 {
				return
			}
			// errors.As digs through the wrapping and the join and returns the first *ArmError.
			var ae *ArmError
			if !errors.As(err, &ae) {
				t.Fatalf("error %v does not contain an *ArmError", err)
			}
			if ae.Arm != tc.badArms[0] {
				t.Errorf("first bad arm = %d, want %d", ae.Arm, tc.badArms[0])
			}
			// Every bad arm shows up in the message, not just the first.
			for _, arm := range tc.badArms {
				if want := fmt.Sprintf("arm %d:", arm); !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not mention %q", err, want)
				}
			}
		})
	}
}

func newEnv(t *testing.T, seed uint64, probs ...float64) *Env {
	t.Helper()
	env, err := NewEnv(probs, NewRNG(seed, StreamEnv))
	if err != nil {
		t.Fatal(err)
	}
	return env
}

func TestNumArmsAndProb(t *testing.T) {
	env := newEnv(t, 1, 0.1, 0.9, 0.5)
	if env.NumArms() != 3 {
		t.Errorf("NumArms() = %d, want 3", env.NumArms())
	}
	for arm, want := range []float64{0.1, 0.9, 0.5} {
		if got := env.Prob(arm); got != want {
			t.Errorf("Prob(%d) = %v, want %v", arm, got, want)
		}
	}
}

func TestNewEnvCopiesItsInput(t *testing.T) {
	probs := []float64{0.1, 0.9}
	env, err := NewEnv(probs, NewRNG(1, StreamEnv))
	if err != nil {
		t.Fatal(err)
	}
	probs[0] = 0.99
	if env.Prob(0) != 0.1 {
		t.Errorf("changing the caller's slice changed the environment: Prob(0) = %v", env.Prob(0))
	}
}

func TestPullReturnsOnlyZeroOrOne(t *testing.T) {
	env := newEnv(t, 1, 0.5)
	for range 1000 {
		if r := env.Pull(0); r != 0 && r != 1 {
			t.Fatalf("Pull returned %v, want 0 or 1", r)
		}
	}
}

func TestPullFrequencyMatchesProbability(t *testing.T) {
	env := newEnv(t, 11, 0, 0.3, 1)
	const n = 20000
	for arm, want := range []float64{0, 0.3, 1} {
		var sum float64
		for range n {
			sum += env.Pull(arm)
		}
		if got := sum / n; math.Abs(got-want) > 0.02 {
			t.Errorf("arm %d paid %.3f of the time, want about %.2f", arm, got, want)
		}
	}
}

func TestPullIsDeterministicForASeed(t *testing.T) {
	a, b := newEnv(t, 9, 0.5, 0.5), newEnv(t, 9, 0.5, 0.5)
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
			arm, p := newEnv(t, 1, tc.probs...).Best()
			if arm != tc.wantArm || p != tc.wantP {
				t.Errorf("Best() = (%d, %v), want (%d, %v)", arm, p, tc.wantArm, tc.wantP)
			}
		})
	}
}
