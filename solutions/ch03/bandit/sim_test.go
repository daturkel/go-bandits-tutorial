package bandit

import (
	"errors"
	"slices"
	"testing"
)

// run is a test helper: t.Helper makes failures point at the caller's line.
func run(t *testing.T, spec string, seed uint64) Result {
	t.Helper()
	env, err := NewEnv([]float64{0.2, 0.5, 0.8}, NewRNG(seed, StreamEnv))
	if err != nil {
		t.Fatal(err)
	}
	pol, err := NewPolicy(spec, env.NumArms(), NewRNG(seed, StreamPolicy))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Run(env, pol, 2000)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestRunIsDeterministic(t *testing.T) {
	for _, spec := range PolicyNames() {
		t.Run(spec, func(t *testing.T) {
			a, b := run(t, spec, 7), run(t, spec, 7)
			if !slices.Equal(a.Pulls, b.Pulls) || a.TotalReward != b.TotalReward {
				t.Fatalf("same seed gave different runs: %v vs %v", a.Pulls, b.Pulls)
			}
		})
	}
}

func TestPoliciesFindTheBestArm(t *testing.T) {
	for _, spec := range PolicyNames() {
		t.Run(spec, func(t *testing.T) {
			res := run(t, spec, 3)
			if best := slices.Max(res.Pulls); res.Pulls[2] != best {
				t.Errorf("arm 2 is best but pulls were %v", res.Pulls)
			}
		})
	}
}

func TestRegretCurveIsMonotone(t *testing.T) {
	res := run(t, "ucb1", 1)
	if !slices.IsSorted(res.RegretCurve) {
		t.Error("cumulative regret must never decrease")
	}
	if last := res.RegretCurve[len(res.RegretCurve)-1]; last != res.Regret {
		t.Errorf("curve ends at %v but Regret is %v", last, res.Regret)
	}
}

// misbehaving always picks an arm that does not exist.
type misbehaving struct{}

func (misbehaving) Name() string        { return "misbehaving" }
func (misbehaving) Select() int         { return 99 }
func (misbehaving) Update(int, float64) {}

func TestRunRejectsOutOfRangeArm(t *testing.T) {
	env, _ := NewEnv([]float64{0.5, 0.5}, NewRNG(1, StreamEnv))
	_, err := Run(env, misbehaving{}, 10)
	if !errors.Is(err, ErrArmOutOfRange) {
		t.Fatalf("error = %v, want ErrArmOutOfRange", err)
	}
}
