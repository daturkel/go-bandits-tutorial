// Package harness runs several policies across many seeds and summarises
// their regret.
package harness

import (
	"banditlab/bandit"
	"banditlab/stat"
)

// Spec describes one comparison.
type Spec struct {
	Scenario string
	Probs    []float64
	Policies []string // policy specs understood by bandit.NewPolicy
	Steps    int
	Seeds    int    // number of independent runs per policy
	BaseSeed uint64 // run i uses seed BaseSeed+i
}

// Curve is one policy's regret averaged over all seeds.
type Curve struct {
	Policy string
	Mean   []float64 // mean cumulative regret after each step
	StdErr []float64 // standard error of that mean
	Final  []float64 // each seed's final regret
}

// Report is the outcome of Compare.
type Report struct {
	Spec   Spec
	Curves []Curve
}

// Compare runs every policy in spec against spec.Seeds independent
// environments and averages the regret curves.
func Compare(spec Spec) (*Report, error) {
	// TASK 5:
	//  - fewer than one step or one seed, or no policies: return an error
	//    (errors.New is fine; the tests only check that there is one).
	//  - for every policy spec, run it once per seed with runOnce, using seed
	//    spec.BaseSeed+i for the i-th run, and collect each run's RegretCurve.
	//    A failing run stops everything with an error naming the policy spec
	//    and the seed and wrapping the cause.
	//  - summarise each policy's runs into a Curve (summarise is given; the
	//    name to pass is the Policy field of any run's Result) and put the
	//    curves in the Report in the same order as spec.Policies.
	// You will need the "errors" and "fmt" packages: add them to the imports.
	return &Report{Spec: spec}, nil
}

func runOnce(spec Spec, policySpec string, seed uint64) (bandit.Result, error) {
	env, err := bandit.NewEnv(spec.Probs, bandit.NewRNG(seed, bandit.StreamEnv))
	if err != nil {
		return bandit.Result{}, err
	}
	pol, err := bandit.NewPolicy(policySpec, env.NumArms(), bandit.NewRNG(seed, bandit.StreamPolicy))
	if err != nil {
		return bandit.Result{}, err
	}
	return bandit.Run(env, pol, spec.Steps)
}

// summarise collapses runs (one regret curve per seed) into a mean curve.
func summarise(name string, runs [][]float64) Curve {
	steps := len(runs[0])
	c := Curve{
		Policy: name,
		Mean:   make([]float64, steps),
		StdErr: make([]float64, steps),
		Final:  make([]float64, len(runs)),
	}
	column := make([]float64, len(runs))
	for t := range steps {
		for i, run := range runs {
			column[i] = run[t]
		}
		c.Mean[t] = stat.Mean(column)
		c.StdErr[t] = stat.StdErr(column)
	}
	for i, run := range runs {
		c.Final[i] = run[steps-1]
	}
	return c
}
