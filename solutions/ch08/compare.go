package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"banditlab/bandit"
	"banditlab/harness"
	"banditlab/stat"
)

// compareCmd runs every policy over many seeds and reports mean final regret,
// optionally writing the full curves as CSV and SVG.
func compareCmd(ctx context.Context, args []string, out, errOut io.Writer) error {
	fs := flag.NewFlagSet("banditsim compare", flag.ContinueOnError)
	scenario := fs.String("scenario", "needle", "test bed: "+strings.Join(bandit.ScenarioNames(), ", "))
	policies := fs.String("policy", "epsgreedy:0.1,epsgreedy:0.01,ucb1,thompson", "comma-separated policy specs")
	steps := fs.Int("steps", 5000, "pulls per run")
	seeds := fs.Int("seeds", 100, "independent runs per policy")
	workers := fs.Int("workers", 0, "parallel runs at once (0 = one per CPU core, 1 = sequential)")
	timeout := fs.Duration("timeout", 0, "give up after this long, e.g. 30s (0 = no limit)")
	progress := fs.Bool("progress", false, "print progress to stderr")
	csvPath := fs.String("csv", "", "write mean regret curves to this CSV file")
	svgPath := fs.String("svg", "", "write a regret chart to this SVG file")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	probs, err := bandit.Scenario(*scenario)
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}
	if *timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	spec := harness.Spec{
		Scenario: *scenario,
		Probs:    probs,
		Policies: strings.Split(*policies, ","),
		Steps:    *steps,
		Seeds:    *seeds,
		BaseSeed: 1,
		Workers:  *workers,
	}
	stopProgress := func() {}
	if *progress {
		var update func(done, total int)
		update, stopProgress = reportProgress(ctx, errOut, 250*time.Millisecond)
		spec.OnProgress = update
	}
	rep, err := harness.Compare(ctx, spec)
	stopProgress() // before anything else is printed
	if err != nil {
		return fmt.Errorf("compare: %w", err)
	}

	fmt.Fprintf(out, "scenario=%s arms=%v steps=%d seeds=%d\n\n", *scenario, probs, *steps, *seeds)
	fmt.Fprintf(out, "%-22s %14s\n", "policy", "final regret")
	for _, c := range rep.Curves {
		fmt.Fprintf(out, "%-22s %8.1f ± %.1f\n", c.Policy, stat.Mean(c.Final), stat.StdErr(c.Final))
	}

	if *csvPath != "" {
		if err := writeFile(*csvPath, func(w io.Writer) error { return rep.WriteCSV(w, 100) }); err != nil {
			return err
		}
	}
	if *svgPath != "" {
		if err := writeFile(*svgPath, rep.WriteSVG); err != nil {
			return err
		}
	}
	return nil
}

// writeFile creates path, fills it with write, and reports the first of the
// write and close errors (a failed Close can mean lost data).
func writeFile(path string, write func(io.Writer) error) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	return write(f)
}

// reportProgress prints "done/total runs" to w every interval, from its own
// goroutine, until stop is called or ctx ends. update is safe to call from
// any goroutine; stop waits for the reporting goroutine to exit, so nothing
// is left running after the command returns. Call stop exactly once.
func reportProgress(ctx context.Context, w io.Writer, interval time.Duration) (update func(done, total int), stop func()) {
	var done, total atomic.Int64
	quit := make(chan struct{})
	finished := make(chan struct{})

	go func() {
		defer close(finished)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fmt.Fprintf(w, "\r%d/%d runs", done.Load(), total.Load())
			case <-quit:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	update = func(d, t int) {
		done.Store(int64(d))
		total.Store(int64(t))
	}
	stop = func() {
		close(quit)
		<-finished
		fmt.Fprintf(w, "\r%d/%d runs\n", done.Load(), total.Load())
	}
	return update, stop
}
