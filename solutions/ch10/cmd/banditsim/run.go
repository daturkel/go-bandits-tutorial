package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"banditlab/internal/bandit"
)

// runCmd simulates one seed of each policy and prints a one-line summary each.
func runCmd(ctx context.Context, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("banditsim run", flag.ContinueOnError)
	scenario := fs.String("scenario", "easy", "test bed: "+strings.Join(bandit.ScenarioNames(), ", "))
	policies := fs.String("policy", "epsgreedy,ucb1,thompson", "comma-separated policy specs: "+strings.Join(bandit.PolicyNames(), ", "))
	steps := fs.Int("steps", 10000, "number of pulls")
	seed := fs.Uint64("seed", 42, "random seed")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	probs, err := bandit.Scenario(*scenario)
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	// Build every environment and policy before printing anything, so a bad
	// spec fails cleanly instead of after a half-written report.
	type entry struct {
		env *bandit.Env
		pol bandit.Policy
	}
	var entries []entry
	for _, spec := range strings.Split(*policies, ",") {
		// A fresh environment per policy, seeded identically, so each policy
		// faces the same starting conditions.
		env, err := bandit.NewEnv(probs, bandit.NewRNG(*seed, bandit.StreamEnv))
		if err != nil {
			return err
		}
		pol, err := bandit.NewPolicy(spec, env.NumArms(), bandit.NewRNG(*seed, bandit.StreamPolicy))
		if err != nil {
			return fmt.Errorf("%w: %w", errUsage, err)
		}
		entries = append(entries, entry{env, pol})
	}

	fmt.Fprintf(out, "scenario=%s arms=%v steps=%d seed=%d\n\n", *scenario, probs, *steps, *seed)
	fmt.Fprintf(out, "%-22s %8s %10s  %s\n", "policy", "reward", "regret", "pulls per arm")
	for _, e := range entries {
		res, err := bandit.Run(ctx, e.env, e.pol, *steps)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%-22s %8.0f %10.1f  %v\n", res.Policy, res.TotalReward, res.Regret, res.Pulls)
	}
	return nil
}
