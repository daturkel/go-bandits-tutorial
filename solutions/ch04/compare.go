package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"banditlab/bandit"
	"banditlab/harness"
	"banditlab/stat"
)

// compareCmd runs every policy over many seeds and reports mean final regret,
// optionally writing the full curves as CSV and SVG.
func compareCmd(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("banditsim compare", flag.ContinueOnError)
	scenario := fs.String("scenario", "needle", "test bed: "+strings.Join(bandit.ScenarioNames(), ", "))
	policies := fs.String("policy", "epsgreedy:0.1,epsgreedy:0.01,ucb1,thompson", "comma-separated policy specs")
	steps := fs.Int("steps", 5000, "pulls per run")
	seeds := fs.Int("seeds", 100, "independent runs per policy")
	csvPath := fs.String("csv", "", "write mean regret curves to this CSV file")
	svgPath := fs.String("svg", "", "write a regret chart to this SVG file")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	probs, err := bandit.Scenario(*scenario)
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}
	rep, err := harness.Compare(harness.Spec{
		Scenario: *scenario,
		Probs:    probs,
		Policies: strings.Split(*policies, ","),
		Steps:    *steps,
		Seeds:    *seeds,
		BaseSeed: 1,
	})
	if err != nil {
		return err
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
