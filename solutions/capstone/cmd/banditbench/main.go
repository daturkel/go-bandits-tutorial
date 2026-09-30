// Command banditbench runs the whole system in one process under load and
// compares policies while the replicas' view of the world is stale.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"banditlab/internal/bandit"
	"banditlab/internal/bench"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, "banditbench:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("banditbench", flag.ContinueOnError)
	fs.SetOutput(stderr)
	scenario := fs.String("scenario", "easy", "the world the users live in: "+strings.Join(bandit.ScenarioNames(), ", "))
	policies := fs.String("policy", "epsgreedy:0.1,ucb1,thompson", "comma-separated policy specs")
	lags := fs.String("lags", "0s,20ms,100ms,400ms", "comma-separated snapshot lags to compare")
	replicas := fs.Int("replicas", 3, "policy replicas")
	workers := fs.Int("c", 12, "concurrent simulated users")
	selections := fs.Int("n", 4000, "selections per run")
	repeats := fs.Int("repeats", 3, "independent runs per setting")
	rewardDelay := fs.Duration("reward-delay", 2*time.Millisecond, "how long a user takes to react before the reward is sent")
	think := fs.Duration("think", 0, "pause between one user and the next")
	seed := fs.Uint64("seed", 1, "random seed")
	if err := fs.Parse(args); err != nil {
		return err
	}
	probs, err := bandit.Scenario(*scenario)
	if err != nil {
		return err
	}
	var ls []time.Duration
	for f := range strings.SplitSeq(*lags, ",") {
		d, err := time.ParseDuration(strings.TrimSpace(f))
		if err != nil {
			return fmt.Errorf("bad lag %q: %w", f, err)
		}
		ls = append(ls, d)
	}
	cfg := bench.Config{
		Policies: strings.Split(*policies, ","), Lags: ls, Replicas: *replicas, Workers: *workers,
		Selections: *selections, Repeats: *repeats, Probs: probs,
		RewardDelay: *rewardDelay, Think: *think, Seed: *seed,
	}
	points, err := bench.Run(ctx, cfg)
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "scenario=%s arms=%v replicas=%d users=%d selections=%d repeats=%d reward-delay=%v\n\n",
		*scenario, probs, *replicas, *workers, *selections, *repeats, *rewardDelay)
	fmt.Fprintf(stdout, "regret per 1000 selections (± standard error), share of selections on the best arm\n\n")
	fmt.Fprintf(stdout, "%-10s", "lag")
	for _, p := range cfg.Policies {
		fmt.Fprintf(stdout, " %22s", p)
	}
	fmt.Fprintln(stdout)
	for i, lag := range ls {
		fmt.Fprintf(stdout, "%-10v", lag)
		for p := range cfg.Policies {
			pt := points[p*len(ls)+i]
			fmt.Fprintf(stdout, "   %6.1f ± %-4.1f (%3.0f%%)", pt.Regret, pt.StdErr, pt.BestShare*100)
		}
		fmt.Fprintln(stdout)
	}
	return nil
}
