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

func TestPullFrequencyMatchesProbability(t *testing.T) {
	env, err := NewEnv([]float64{0.3}, NewRNG(11, StreamEnv))
	if err != nil {
		t.Fatal(err)
	}
	const n = 20000
	var sum float64
	for range n {
		sum += env.Pull(0)
	}
	if got := sum / n; math.Abs(got-0.3) > 0.02 {
		t.Errorf("empirical rate %.3f, want about 0.3", got)
	}
}

func TestBest(t *testing.T) {
	env, _ := NewEnv([]float64{0.1, 0.9, 0.9, 0.4}, NewRNG(1, StreamEnv))
	if arm, p := env.Best(); arm != 1 || p != 0.9 {
		t.Errorf("Best() = (%d, %v), want (1, 0.9)", arm, p)
	}
}
