package bandit

import (
	"slices"
	"testing"
)

func run(seed uint64, eps float64) Result {
	env := NewEnv([]float64{0.2, 0.5, 0.8}, NewRNG(seed, StreamEnv))
	pol := NewEpsilonGreedy(env.NumArms(), eps, NewRNG(seed, StreamPolicy))
	return Run(env, pol, 2000)
}

func TestRunIsDeterministic(t *testing.T) {
	a, b := run(7, 0.1), run(7, 0.1)
	if !slices.Equal(a.Pulls, b.Pulls) || a.TotalReward != b.TotalReward {
		t.Fatalf("same seed gave different runs: %v vs %v", a.Pulls, b.Pulls)
	}
}

func TestExploringBeatsNeverExploring(t *testing.T) {
	// Averaged over a few seeds so one unlucky start cannot decide the test.
	var greedy, explore float64
	for seed := uint64(1); seed <= 10; seed++ {
		greedy += run(seed, 0).Regret
		explore += run(seed, 0.1).Regret
	}
	if explore >= greedy {
		t.Fatalf("epsilon=0.1 regret %.1f should be below epsilon=0 regret %.1f", explore, greedy)
	}
}
