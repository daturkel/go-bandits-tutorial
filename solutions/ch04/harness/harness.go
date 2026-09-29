// Package harness runs several policies across many seeds and summarises
// their regret.
package harness

import (
	"errors"
	"fmt"

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
	if spec.Steps < 1 || spec.Seeds < 1 {
		return nil, errors.New("harness: steps and seeds must be positive")
	}
	if len(spec.Policies) == 0 {
		return nil, errors.New("harness: no policies to compare")
	}
	rep := &Report{Spec: spec}
	for _, policySpec := range spec.Policies {
		runs := make([][]float64, 0, spec.Seeds)
		var name string
		for i := range spec.Seeds {
			seed := spec.BaseSeed + uint64(i)
			res, err := runOnce(spec, policySpec, seed)
			if err != nil {
				return nil, fmt.Errorf("policy %q, seed %d: %w", policySpec, seed, err)
			}
			name = res.Policy
			runs = append(runs, res.RegretCurve)
		}
		rep.Curves = append(rep.Curves, summarise(name, runs))
	}
	return rep, nil
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
