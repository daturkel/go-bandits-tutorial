// Package harness runs several policies across many seeds and summarises
// their regret.
package harness

import (
	"errors"
	"fmt"
	"sync"

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

// Compare runs every policy against spec.Seeds independent environments and
// averages the regret curves. Every (policy, seed) run gets its own goroutine;
// the report is the same as CompareSequential's, whatever order they finish in.
func Compare(spec Spec) (*Report, error) {
	if err := spec.validate(); err != nil {
		return nil, err
	}

	// An outcome carries the indexes of its job, and is stored at those
	// indexes, so the report does not depend on which goroutine finishes first.
	type outcome struct {
		policy, seed int
		res          bandit.Result
		err          error
	}
	results := make(chan outcome)

	var wg sync.WaitGroup
	for p, policySpec := range spec.Policies {
		for i := range spec.Seeds {
			wg.Add(1)
			go func() {
				defer wg.Done()
				res, err := runOnce(spec, policySpec, spec.BaseSeed+uint64(i))
				results <- outcome{policy: p, seed: i, res: res, err: err}
			}()
		}
	}
	// Close the channel once every sender is done, so the loop below ends.
	go func() {
		wg.Wait()
		close(results)
	}()

	runs := make([][][]float64, len(spec.Policies)) // runs[policy][seed]
	names := make([]string, len(spec.Policies))
	errs := make([][]error, len(spec.Policies))
	for p := range spec.Policies {
		runs[p] = make([][]float64, spec.Seeds)
		errs[p] = make([]error, spec.Seeds)
	}
	for o := range results { // receives until the channel is closed
		if o.err != nil {
			errs[o.policy][o.seed] = o.err
			continue
		}
		names[o.policy] = o.res.Policy
		runs[o.policy][o.seed] = o.res.RegretCurve
	}

	// Report the failure that comes first in (policy, seed) order, the one
	// CompareSequential would have stopped at.
	for p := range errs {
		for i, err := range errs[p] {
			if err != nil {
				return nil, fmt.Errorf("policy %q, seed %d: %w", spec.Policies[p], spec.BaseSeed+uint64(i), err)
			}
		}
	}
	rep := &Report{Spec: spec}
	for p := range runs {
		rep.Curves = append(rep.Curves, summarise(names[p], runs[p]))
	}
	return rep, nil
}

// CompareSequential runs the same jobs one after another. It is the reference
// Compare must match, and the baseline for its benchmark.
func CompareSequential(spec Spec) (*Report, error) {
	if err := spec.validate(); err != nil {
		return nil, err
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

// validate rejects specs that cannot produce a report.
func (s Spec) validate() error {
	if s.Steps < 1 || s.Seeds < 1 {
		return errors.New("harness: steps and seeds must be positive")
	}
	if len(s.Policies) == 0 {
		return errors.New("harness: no policies to compare")
	}
	return nil
}
