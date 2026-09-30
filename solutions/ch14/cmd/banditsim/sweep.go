package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"banditlab/internal/bandit"
	"banditlab/internal/harness"
)

// sweepCmd simulates several replicas that share a stream of requests and
// sync their counts every so often, and reports how regret depends on the
// sync interval.
func sweepCmd(ctx context.Context, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("banditsim sweep", flag.ContinueOnError)
	scenario := fs.String("scenario", "needle", "test bed: "+strings.Join(bandit.ScenarioNames(), ", "))
	policies := fs.String("policy", "epsgreedy:0.1,ucb1,thompson", "comma-separated policy specs")
	replicas := fs.Int("replicas", 4, "number of replicas sharing the traffic")
	intervals := fs.String("intervals", "0,1000,100,10,1", "comma-separated sync intervals in selections; 0 means never")
	steps := fs.Int("steps", 5000, "selections per run")
	seeds := fs.Int("seeds", 40, "independent runs per setting")
	csvPath := fs.String("csv", "", "write the results to this CSV file")
	svgPath := fs.String("svg", "", "write a chart to this SVG file")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}
	probs, err := bandit.Scenario(*scenario)
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}
	var ivs []int
	for f := range strings.SplitSeq(*intervals, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			return fmt.Errorf("%w: bad interval %q", errUsage, f)
		}
		ivs = append(ivs, n)
	}

	spec := harness.SyncSpec{
		Scenario: *scenario, Probs: probs, Policies: strings.Split(*policies, ","),
		Replicas: *replicas, Steps: *steps, Seeds: *seeds, BaseSeed: 1, Intervals: ivs,
	}
	points, err := harness.SweepSync(ctx, spec)
	if err != nil {
		return fmt.Errorf("sweep: %w", err)
	}

	fmt.Fprintf(out, "scenario=%s arms=%v replicas=%d steps=%d seeds=%d\n\n", *scenario, probs, *replicas, *steps, *seeds)
	fmt.Fprintf(out, "%-14s", "sync every")
	for _, p := range spec.Policies {
		fmt.Fprintf(out, " %14s", p)
	}
	fmt.Fprintln(out)
	for i, iv := range ivs {
		name := strconv.Itoa(iv)
		if iv == 0 {
			name = "never"
		}
		fmt.Fprintf(out, "%-14s", name)
		for p := range spec.Policies {
			fmt.Fprintf(out, " %14.1f", points[p*len(ivs)+i].Regret)
		}
		fmt.Fprintln(out)
	}

	if *csvPath != "" {
		if err := writeFile(*csvPath, func(w io.Writer) error { return harness.WriteSyncCSV(w, points) }); err != nil {
			return err
		}
	}
	if *svgPath != "" {
		if err := writeFile(*svgPath, func(w io.Writer) error { return harness.WriteSyncSVG(w, spec, points) }); err != nil {
			return err
		}
	}
	return nil
}
