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
	epsilon := flag.Float64("eps", 0.1, "exploration rate for epsilon-greedy")
	steps := flag.Int("steps", 10000, "number of pulls")
	seed := flag.Uint64("seed", 42, "random seed")
	flag.Parse()

	probs, ok := bandit.Scenario(*scenario)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown scenario %q\n", *scenario)
		os.Exit(2)
	}

	env := bandit.NewEnv(probs, bandit.NewRNG(*seed, bandit.StreamEnv))
	pol := bandit.NewEpsilonGreedy(env.NumArms(), *epsilon, bandit.NewRNG(*seed, bandit.StreamPolicy))
	res := bandit.Run(env, pol, *steps)

	fmt.Printf("scenario=%s arms=%v policy=epsilon-greedy(%.2f) steps=%d seed=%d\n\n",
		*scenario, probs, *epsilon, *steps, *seed)
	fmt.Printf("%8s  %10s\n", "step", "regret")
	for n := 10; n <= res.Steps; n *= 10 {
		fmt.Printf("%8d  %10.1f\n", n, res.RegretCurve[n-1])
	}
	fmt.Printf("\npulls per arm: %v\n", res.Pulls)
	fmt.Printf("total reward:  %.0f\n", res.TotalReward)
	fmt.Printf("final regret:  %.1f\n", res.Regret)
}
