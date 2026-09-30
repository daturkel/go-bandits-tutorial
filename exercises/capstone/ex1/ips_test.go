package ex1

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestIPSByHand(t *testing.T) {
	// Three decisions made uniformly at random over two arms (propensity 0.5).
	log := []Logged{{0, 1, 0.5}, {1, 0, 0.5}, {1, 1, 0.5}}
	// A new policy that always picks arm 1 would have earned: only decisions
	// on arm 1 count, each weighted 1/0.5 = 2, so (0*2 + 1*2) / 3.
	got, ok := IPS(log, []float64{0, 1})
	if !ok || math.Abs(got-2.0/3.0) > 1e-12 {
		t.Errorf("IPS = %v, %v; want 2/3", got, ok)
	}
	// The logging policy itself: the weights are all 1, so this is the plain mean.
	got, ok = IPS(log, []float64{0.5, 0.5})
	if !ok || math.Abs(got-2.0/3.0) > 1e-12 {
		t.Errorf("IPS of the logging policy = %v, %v; want the mean, 2/3", got, ok)
	}
}

func TestIPSRecoversTheTrueValue(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	probs := []float64{0.2, 0.5, 0.8}
	logging := []float64{0.5, 0.3, 0.2} // mostly the worse arms
	cumulative := []float64{0.5, 0.8, 1.0}
	var log []Logged
	for range 200_000 {
		u, arm := rng.Float64(), 0
		for arm < 2 && u >= cumulative[arm] {
			arm++
		}
		r := 0.0
		if rng.Float64() < probs[arm] {
			r = 1
		}
		log = append(log, Logged{arm, r, logging[arm]})
	}
	// A policy that plays the best arm 90% of the time is worth 0.9*0.8+0.05*0.5+0.05*0.2 = 0.755.
	got, ok := IPS(log, []float64{0.05, 0.05, 0.9})
	if !ok || math.Abs(got-0.755) > 0.01 {
		t.Errorf("IPS = %v, %v; want about 0.755", got, ok)
	}
}

func TestIPSRejectsBadLogs(t *testing.T) {
	good := []Logged{{0, 1, 0.5}}
	for name, tc := range map[string]struct {
		log    []Logged
		target []float64
	}{
		"empty log":         {nil, []float64{1, 0}},
		"zero propensity":   {[]Logged{{0, 1, 0}}, []float64{1, 0}},
		"propensity over 1": {[]Logged{{0, 1, 1.5}}, []float64{1, 0}},
		"arm outside":       {[]Logged{{2, 1, 0.5}}, []float64{1, 0}},
		"negative arm":      {[]Logged{{-1, 1, 0.5}}, []float64{1, 0}},
	} {
		if got, ok := IPS(tc.log, tc.target); ok {
			t.Errorf("%s: IPS = %v, ok; want not ok", name, got)
		}
	}
	if _, ok := IPS(good, []float64{1, 0}); !ok {
		t.Error("a valid log was rejected")
	}
}
