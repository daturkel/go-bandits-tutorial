// Command banditload drives a running banditd with simulated users.
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
	"banditlab/internal/load"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, "banditload:", err)
			os.Exit(2)
		}
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("banditload", flag.ContinueOnError)
	fs.SetOutput(stderr)
	url := fs.String("url", "http://localhost:8080", "base `url` of banditd; a comma-separated list spreads the users over several replicas")
	workers := fs.Int("c", 16, "concurrent simulated users")
	dur := fs.Duration("d", 10*time.Second, "how long to run")
	rate := fs.Float64("rate", 0, "selections per second (open loop); 0 sends as fast as the server answers (closed loop)")
	scenario := fs.String("scenario", "easy", "the world the users live in: "+strings.Join(bandit.ScenarioNames(), ", "))
	seed := fs.Uint64("seed", 1, "random seed")
	fraction := fs.Float64("reward-fraction", 1, "fraction of selections that get a reward call")
	delay := fs.Duration("feedback-delay", 0, "pause between a selection and its reward")
	if err := fs.Parse(args); err != nil {
		return err
	}
	probs, err := bandit.Scenario(*scenario)
	if err != nil {
		return err
	}

	var urls []string
	for u := range strings.SplitSeq(*url, ",") {
		if u = strings.TrimRight(strings.TrimSpace(u), "/"); u != "" {
			urls = append(urls, u)
		}
	}
	if len(urls) == 0 {
		return fmt.Errorf("-url is empty")
	}

	rep, err := load.Run(ctx, load.Config{
		BaseURLs:       urls,
		Workers:        *workers,
		Elapsed:        *dur,
		Rate:           *rate,
		Probs:          probs,
		Seed:           *seed,
		RewardFraction: *fraction,
		FeedbackDelay:  *delay,
	})
	if err != nil {
		return err
	}
	rep.Write(stdout)
	return nil
}
