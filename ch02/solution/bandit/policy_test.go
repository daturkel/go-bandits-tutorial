package bandit

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

// Compile-time checks: these lines stop compiling if a type stops satisfying Policy.
var (
	_ Policy = (*EpsilonGreedy)(nil)
	_ Policy = (*UCB1)(nil)
)

func TestPolicyHasExactlyTheMethodsRunNeeds(t *testing.T) {
	typ := reflect.TypeOf((*Policy)(nil)).Elem()
	var names []string
	for i := range typ.NumMethod() {
		names = append(names, typ.Method(i).Name)
	}
	if want := []string{"Name", "Select", "Update"}; !slices.Equal(names, want) {
		t.Errorf("Policy has methods %v, want %v", names, want)
	}
}

func TestNewPolicyBuildsByName(t *testing.T) {
	for name, want := range map[string]string{"epsgreedy": "epsilon-greedy(0.20)", "ucb1": "ucb1"} {
		pol, err := NewPolicy(name, 3, 0.2, NewRNG(1, StreamPolicy))
		if err != nil || pol == nil {
			t.Fatalf("NewPolicy(%q) = %v, %v", name, pol, err)
		}
		if pol.Name() != want {
			t.Errorf("NewPolicy(%q).Name() = %q, want %q", name, pol.Name(), want)
		}
		if arm := pol.Select(); arm < 0 || arm > 2 {
			t.Errorf("%s selected arm %d of 3", name, arm)
		}
	}
}

func TestNewPolicyRejectsUnknownNames(t *testing.T) {
	pol, err := NewPolicy("nope", 3, 0.1, NewRNG(1, StreamPolicy))
	if err == nil {
		t.Fatal("an unknown name must be an error")
	}
	if pol != nil {
		// A typed nil pointer wrapped in the interface is not == nil, and
		// callers that check `pol != nil` would then use it and crash.
		t.Errorf("Policy = %#v, want a true nil interface value", pol)
	}
	if msg := err.Error(); !strings.Contains(msg, "nope") {
		t.Errorf("error %q should mention the name", msg)
	}
}
