package bandit

import (
	"math"
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

func TestRunRecordsThePolicyName(t *testing.T) {
	if got := run(t, "ucb1", 1).Policy; got != "ucb1" {
		t.Errorf("Result.Policy = %q, want ucb1", got)
	}
}

func TestRunAccountingForEveryPolicy(t *testing.T) {
	for _, name := range PolicyNames() {
		res := run(t, name, 3)
		total, gaps := 0, 0.0
		for arm, n := range res.Pulls {
			total += n
			gaps += float64(n) * (0.8 - []float64{0.2, 0.5, 0.8}[arm])
		}
		if total != 2000 || math.Abs(res.Regret-gaps) > 1e-6 || !slices.IsSorted(res.RegretCurve) {
			t.Errorf("%s: pulls=%v (sum %d) regret=%v implied=%v", name, res.Pulls, total, res.Regret, gaps)
		}
	}
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
