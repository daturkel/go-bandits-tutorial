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
// averages the regret curves. Each (policy, seed) run gets its own goroutine.
func Compare(spec Spec) (*Report, error) {
	if err := spec.validate(); err != nil {
		return nil, err
	}

	// A job is identified by its indexes, and its outcome is stored at the
	// same indexes, so the report does not depend on which goroutine
	// finishes first.
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

	runs := newRuns(spec)
	names := make([]string, len(spec.Policies))
	errs := make([][]error, len(spec.Policies))
	for p := range errs {
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
	if err := firstError(spec, errs); err != nil {
		return nil, err
	}
	return buildReport(spec, names, runs), nil
}

// CompareSequential is Compare without goroutines. It exists as the reference
// the parallel version must match, and as the baseline for benchmarks.
func CompareSequential(spec Spec) (*Report, error) {
	if err := spec.validate(); err != nil {
		return nil, err
	}
	runs := newRuns(spec)
	names := make([]string, len(spec.Policies))
	errs := make([][]error, len(spec.Policies))
	for p, policySpec := range spec.Policies {
		errs[p] = make([]error, spec.Seeds)
		for i := range spec.Seeds {
			res, err := runOnce(spec, policySpec, spec.BaseSeed+uint64(i))
			if err != nil {
				errs[p][i] = err
				continue
			}
			names[p] = res.Policy
			runs[p][i] = res.RegretCurve
		}
	}
	if err := firstError(spec, errs); err != nil {
		return nil, err
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

// firstError reports the failed run that comes first in (policy, seed)
// order, so the error does not depend on goroutine scheduling.
func firstError(spec Spec, errs [][]error) error {
	for p, row := range errs {
		for i, err := range row {
			if err != nil {
				return fmt.Errorf("policy %q, seed %d: %w", spec.Policies[p], spec.BaseSeed+uint64(i), err)
			}
		}
	}
	return nil
}

func buildReport(spec Spec, names []string, runs [][][]float64) *Report {
	rep := &Report{Spec: spec}
	for p := range runs {
		rep.Curves = append(rep.Curves, summarise(names[p], runs[p]))
	}
	return rep
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
