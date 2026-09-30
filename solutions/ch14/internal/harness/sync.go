package harness

import (
	"context"
	"errors"
	"fmt"

	"banditlab/internal/bandit"
	"banditlab/internal/pool"
	"banditlab/internal/stat"
)

// SyncSpec describes a replica experiment: several copies of one policy
// share a stream of requests and a single true state of the world, and
// exchange what they have learned only every so often.
type SyncSpec struct {
	Scenario string
	Probs    []float64
	Policies []string
	Replicas int
	Steps    int
	Seeds    int
	BaseSeed uint64

	// Intervals lists the sync intervals to try, in selections. Every
	// interval steps, each replica's policy is rebuilt from the totals of
	// all replicas. Zero means never sync: each replica learns only from its
	// own requests.
	Intervals []int

	Workers int // as in Spec
}

// SyncPoint is the result for one policy at one sync interval.
type SyncPoint struct {
	Policy   string
	Interval int
	Regret   float64 // mean final regret over the seeds
	StdErr   float64
}

// SweepSync runs the experiment for every policy and interval. Requests go
// to the replicas round-robin; every reward is recorded in shared totals
// immediately, but a replica sees them only at the next sync.
func SweepSync(ctx context.Context, spec SyncSpec) ([]SyncPoint, error) {
	if spec.Replicas < 1 || spec.Steps < 1 || spec.Seeds < 1 {
		return nil, errors.New("harness: replicas, steps and seeds must be positive")
	}
	if len(spec.Policies) == 0 || len(spec.Intervals) == 0 {
		return nil, errors.New("harness: need at least one policy and one interval")
	}
	for _, iv := range spec.Intervals {
		if iv < 0 {
			return nil, fmt.Errorf("harness: sync interval %d is negative", iv)
		}
	}

	type job struct{ policy, interval, seed int }
	var jobs []job
	for p := range spec.Policies {
		for i := range spec.Intervals {
			for s := range spec.Seeds {
				jobs = append(jobs, job{p, i, s})
			}
		}
	}
	finals, err := pool.Run(ctx, pool.Options{Workers: spec.Workers}, jobs,
		func(ctx context.Context, j job) (float64, error) {
			regret, err := runReplicas(ctx, spec, spec.Policies[j.policy], spec.Intervals[j.interval], spec.BaseSeed+uint64(j.seed))
			if err != nil {
				return 0, fmt.Errorf("policy %q, interval %d, seed %d: %w", spec.Policies[j.policy], spec.Intervals[j.interval], j.seed, err)
			}
			return regret, nil
		})
	if err != nil {
		return nil, err
	}

	var points []SyncPoint
	for p, name := range spec.Policies {
		for i, iv := range spec.Intervals {
			var col []float64
			for k, j := range jobs {
				if j.policy == p && j.interval == i {
					col = append(col, finals[k])
				}
			}
			points = append(points, SyncPoint{Policy: name, Interval: iv, Regret: stat.Mean(col), StdErr: stat.StdErr(col)})
		}
	}
	return points, nil
}

// runReplicas is one run: it returns the expected regret of all choices made.
func runReplicas(ctx context.Context, spec SyncSpec, policySpec string, interval int, seed uint64) (float64, error) {
	env, err := bandit.NewEnv(spec.Probs, bandit.NewRNG(seed, bandit.StreamEnv))
	if err != nil {
		return 0, err
	}
	replicas := make([]bandit.Policy, spec.Replicas)
	for r := range replicas {
		replicas[r], err = bandit.NewPolicy(policySpec, env.NumArms(), bandit.NewRNG(seed, bandit.StreamPolicy+uint64(r)))
		if err != nil {
			return 0, err
		}
		if _, ok := replicas[r].(bandit.Restorer); !ok {
			return 0, fmt.Errorf("policy %s cannot be restored from totals", replicas[r].Name())
		}
	}

	shared := make([]bandit.ArmTotals, env.NumArms())
	for i := range shared {
		shared[i].Arm = i
	}
	_, best := env.Best()
	regret := 0.0
	for t := range spec.Steps {
		if t%1024 == 0 && ctx.Err() != nil {
			return 0, ctx.Err()
		}
		pol := replicas[t%spec.Replicas]
		arm := pol.Select()
		reward := env.Pull(arm)
		regret += best - env.Prob(arm)
		shared[arm].Pulls++
		shared[arm].RewardSum += reward
		pol.Update(arm, reward)

		if interval > 0 && (t+1)%interval == 0 {
			for _, r := range replicas {
				if err := r.(bandit.Restorer).Restore(shared); err != nil {
					return 0, err
				}
			}
		}
	}
	return regret, nil
}
