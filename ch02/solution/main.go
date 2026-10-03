// This file comes with the course, and later chapters may replace it with a
// new version. Keep your own code in other files.

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"banditlab/bandit"
)

func main() {
	scenario := flag.String("scenario", "easy", "test bed: "+strings.Join(bandit.ScenarioNames(), ", "))
	policies := flag.String("policy", "epsgreedy,ucb1", "comma-separated policies: "+strings.Join(bandit.PolicyNames(), ", "))
	epsilon := flag.Float64("eps", 0.1, "exploration rate for epsilon-greedy")
	steps := flag.Int("steps", 10000, "number of pulls")
	seed := flag.Uint64("seed", 42, "random seed")
	flag.Parse()

	probs, ok := bandit.Scenario(*scenario)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown scenario %q\n", *scenario)
		os.Exit(2)
	}

	fmt.Printf("scenario=%s arms=%v steps=%d seed=%d\n\n", *scenario, probs, *steps, *seed)
	fmt.Printf("%-22s %8s %10s  %s\n", "policy", "reward", "regret", "pulls per arm")
	for _, name := range strings.Split(*policies, ",") {
		// A fresh environment per policy, seeded identically, so each policy
		// faces the same starting conditions.
		env := bandit.NewEnv(probs, bandit.NewRNG(*seed, bandit.StreamEnv))
		pol, err := bandit.NewPolicy(name, env.NumArms(), *epsilon, bandit.NewRNG(*seed, bandit.StreamPolicy))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		res := bandit.Run(env, pol, *steps)
		fmt.Printf("%-22s %8.0f %10.1f  %v\n", res.Policy, res.TotalReward, res.Regret, res.Pulls)
	}
}
