package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunPrintsATable(t *testing.T) {
	var out bytes.Buffer
	err := run(context.Background(), []string{
		"-policy", "thompson,ucb1", "-lags", "0s,10ms", "-n", "200", "-repeats", "1", "-c", "4", "-replicas", "2",
	}, &out, &out)
	if err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	text := out.String()
	for _, want := range []string{"replicas=2", "thompson", "ucb1", "0s", "10ms", "±"} {
		if !strings.Contains(text, want) {
			t.Errorf("output lacks %q:\n%s", want, text)
		}
	}
	// A header, a caption, a column line and one row per lag.
	if rows := strings.Count(text, "%)"); rows != 4 {
		t.Errorf("found %d result cells, want 4 (2 lags x 2 policies):\n%s", rows, text)
	}
}

func TestRunRejectsBadInput(t *testing.T) {
	for _, args := range [][]string{
		{"-lags", "soon"},
		{"-scenario", "nope"},
		{"-policy", "nope", "-n", "10"},
		{"-replicas", "0"},
	} {
		var out bytes.Buffer
		if err := run(context.Background(), args, &out, &out); err == nil {
			t.Errorf("run(%v) succeeded, want an error", args)
		}
	}
}
