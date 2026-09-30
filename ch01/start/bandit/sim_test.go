package bandit

import (
	"math"
	"slices"
	"testing"
)

func run(seed uint64, eps float64) Result {
	env := NewEnv([]float64{0.2, 0.5, 0.8}, NewRNG(seed, StreamEnv))
	pol := NewEpsilonGreedy(env.NumArms(), eps, NewRNG(seed, StreamPolicy))
	return Run(env, pol, 2000)
}

func TestRunAccounting(t *testing.T) {
	res := run(3, 0.1)
	if res.Steps != 2000 || len(res.RegretCurve) != 2000 {
		t.Fatalf("Steps=%d, len(RegretCurve)=%d; want 2000 for both", res.Steps, len(res.RegretCurve))
	}
	total := 0
	gaps := 0.0
	for arm, n := range res.Pulls {
		total += n
		gaps += float64(n) * (0.8 - []float64{0.2, 0.5, 0.8}[arm])
	}
	if total != 2000 {
		t.Errorf("Pulls %v add up to %d, want 2000", res.Pulls, total)
	}
	if math.Abs(res.Regret-gaps) > 1e-6 {
		t.Errorf("Regret = %v, but the pulls imply %v", res.Regret, gaps)
	}
	if math.Abs(res.RegretCurve[1999]-res.Regret) > 1e-9 {
		t.Errorf("the curve ends at %v, Regret is %v", res.RegretCurve[1999], res.Regret)
	}
	if !slices.IsSorted(res.RegretCurve) {
		t.Error("cumulative regret must never go down")
	}
	if res.TotalReward < 0 || res.TotalReward > 2000 {
		t.Errorf("TotalReward = %v", res.TotalReward)
	}
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
