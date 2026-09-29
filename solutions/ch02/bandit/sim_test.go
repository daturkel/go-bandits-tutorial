package bandit

import (
	"slices"
	"testing"
)

func run(t *testing.T, name string, seed uint64) Result {
	t.Helper()
	env := NewEnv([]float64{0.2, 0.5, 0.8}, NewRNG(seed, StreamEnv))
	pol, err := NewPolicy(name, env.NumArms(), 0.1, NewRNG(seed, StreamPolicy))
	if err != nil {
		t.Fatal(err)
	}
	return Run(env, pol, 2000)
}

func TestRunIsDeterministic(t *testing.T) {
	for _, name := range PolicyNames() {
		a, b := run(t, name, 7), run(t, name, 7)
		if !slices.Equal(a.Pulls, b.Pulls) || a.TotalReward != b.TotalReward {
			t.Fatalf("%s: same seed gave different runs: %v vs %v", name, a.Pulls, b.Pulls)
		}
	}
}

func TestPoliciesFindTheBestArm(t *testing.T) {
	for _, name := range PolicyNames() {
		res := run(t, name, 3)
		if best := slices.Max(res.Pulls); res.Pulls[2] != best {
			t.Errorf("%s: arm 2 is best but pulls were %v", name, res.Pulls)
		}
	}
}
