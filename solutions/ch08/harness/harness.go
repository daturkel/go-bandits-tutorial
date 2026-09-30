// Package harness runs several policies across many seeds and summarises
// their regret.
package harness

import (
	"context"
	"errors"
	"fmt"

	"banditlab/bandit"
	"banditlab/pool"
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

	// Workers bounds how many runs execute at once. Zero means one per CPU
	// core; 1 runs them one at a time.
	Workers int

	// OnProgress, if set, is called after each finished run with the number
	// done so far and the total. Calls never overlap.
	OnProgress func(done, total int)
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
// averages the regret curves. The runs execute on a bounded pool of
// goroutines (see Spec.Workers); the report does not depend on how many.
//
// If ctx is cancelled or times out, Compare stops promptly and returns
// ctx.Err(). If any run fails, the remaining runs are cancelled and the error
// of the earliest failing (policy, seed) is returned.
func Compare(ctx context.Context, spec Spec) (*Report, error) {
	if err := spec.validate(); err != nil {
		return nil, err
	}

	type job struct{ policy, seed int }
	var jobs []job
	for p := range spec.Policies {
		for i := range spec.Seeds {
			jobs = append(jobs, job{p, i})
		}
	}

	results, err := pool.Run(ctx, pool.Options{Workers: spec.Workers, OnDone: spec.OnProgress}, jobs,
		func(ctx context.Context, j job) (bandit.Result, error) {
			return runOnce(ctx, spec, spec.Policies[j.policy], spec.BaseSeed+uint64(j.seed))
		})
	if err != nil {
		return nil, err
	}

	// Results come back in job order, so slot k belongs to jobs[k].
	runs := newRuns(spec)
	names := make([]string, len(spec.Policies))
	for k, j := range jobs {
		runs[j.policy][j.seed] = results[k].RegretCurve
		names[j.policy] = results[k].Policy
	}
	return buildReport(spec, names, runs), nil
}

func (s Spec) validate() error {
	if s.Steps < 1 || s.Seeds < 1 {
		return errors.New("harness: steps and seeds must be positive")
	}
	if len(s.Policies) == 0 {
		return errors.New("harness: no policies to compare")
	}
	return nil
}

// newRuns allocates runs[policy][seed], to be filled with regret curves.
func newRuns(spec Spec) [][][]float64 {
	runs := make([][][]float64, len(spec.Policies))
	for p := range runs {
		runs[p] = make([][]float64, spec.Seeds)
	}
	return runs
}

func buildReport(spec Spec, names []string, runs [][][]float64) *Report {
	rep := &Report{Spec: spec}
	for p := range runs {
		rep.Curves = append(rep.Curves, summarise(names[p], runs[p]))
	}
	return rep
}

// runOnce simulates one policy on one seed. Errors say which run failed.
func runOnce(ctx context.Context, spec Spec, policySpec string, seed uint64) (bandit.Result, error) {
	fail := func(err error) (bandit.Result, error) {
		return bandit.Result{}, fmt.Errorf("policy %q, seed %d: %w", policySpec, seed, err)
	}
	env, err := bandit.NewEnv(spec.Probs, bandit.NewRNG(seed, bandit.StreamEnv))
	if err != nil {
		return fail(err)
	}
	pol, err := bandit.NewPolicy(policySpec, env.NumArms(), bandit.NewRNG(seed, bandit.StreamPolicy))
	if err != nil {
		return fail(err)
	}
	res, err := bandit.Run(ctx, env, pol, spec.Steps)
	if err != nil {
		return fail(err)
	}
	return res, nil
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
