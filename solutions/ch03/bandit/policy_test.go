package bandit

import (
	"errors"
	"strconv"
	"testing"
)

// Compile-time checks: these lines stop compiling if a type stops satisfying Policy.
var (
	_ Policy = (*EpsilonGreedy)(nil)
	_ Policy = (*UCB1)(nil)
)

func TestNewPolicy(t *testing.T) {
	tests := []struct {
		spec     string
		wantName string
		wantErr  error
	}{
		{spec: "ucb1", wantName: "ucb1"},
		{spec: "epsgreedy", wantName: "epsilon-greedy(0.10)"},
		{spec: "epsgreedy:0.05", wantName: "epsilon-greedy(0.05)"},
		{spec: "epsgreedy:1.5", wantErr: ErrInvalidEpsilon},
		{spec: "thompson", wantErr: ErrUnknownPolicy},
		{spec: "", wantErr: ErrUnknownPolicy},
	}
	for _, tc := range tests {
		t.Run(tc.spec, func(t *testing.T) {
			pol, err := NewPolicy(tc.spec, 3, NewRNG(1, StreamPolicy))
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want one wrapping %v", err, tc.wantErr)
			}
			if err == nil && pol.Name() != tc.wantName {
				t.Errorf("Name() = %q, want %q", pol.Name(), tc.wantName)
			}
		})
	}
}

func TestNewPolicyBadEpsilonText(t *testing.T) {
	_, err := NewPolicy("epsgreedy:lots", 3, NewRNG(1, StreamPolicy))
	// The strconv error is still reachable through our added context.
	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		t.Fatalf("error %v does not wrap a *strconv.NumError", err)
	}
	if numErr.Num != "lots" {
		t.Errorf("NumError.Num = %q, want %q", numErr.Num, "lots")
	}
}

func TestConstructorsRejectZeroArms(t *testing.T) {
	if _, err := NewUCB1(0); !errors.Is(err, ErrNoArms) {
		t.Errorf("NewUCB1(0) error = %v, want ErrNoArms", err)
	}
	if _, err := NewEpsilonGreedy(0, 0.1, NewRNG(1, StreamPolicy)); !errors.Is(err, ErrNoArms) {
		t.Errorf("NewEpsilonGreedy(0) error = %v, want ErrNoArms", err)
	}
}

// Tie-breaking must be random among equal arms, otherwise the first arm is
// favoured before any data has arrived.
func TestGreedyBreaksTiesAcrossAllArms(t *testing.T) {
	pol, _ := NewEpsilonGreedy(4, 0, NewRNG(5, StreamPolicy))
	seen := map[int]bool{}
	for range 200 {
		seen[pol.Select()] = true
	}
	if len(seen) != 4 {
		t.Errorf("greedy tie-break only ever chose arms %v", seen)
	}
}

// A failed constructor must give back a true nil Policy, not an interface
// wrapping a nil pointer (which would compare != nil).
func TestNewPolicyFailureReturnsNilInterface(t *testing.T) {
	pol, err := NewPolicy("epsgreedy:5", 3, NewRNG(1, StreamPolicy))
	if err == nil {
		t.Fatal("expected an error")
	}
	if pol != nil {
		t.Errorf("Policy = %#v, want a nil interface", pol)
	}
}
