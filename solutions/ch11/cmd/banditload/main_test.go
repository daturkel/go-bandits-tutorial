package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"banditlab/internal/bandit"
	"banditlab/internal/server"
	"banditlab/internal/store"
)

func TestRunAgainstARealServer(t *testing.T) {
	pol, err := bandit.NewSnapshotPolicy("thompson", 3, bandit.NewRNG(1, bandit.StreamPolicy))
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(server.New(pol, store.NewMemory(3), slog.New(slog.NewTextHandler(io.Discard, nil))).Handler())
	defer ts.Close()

	var out bytes.Buffer
	args := []string{"-url", ts.URL + "/", "-c", "2", "-d", "200ms", "-scenario", "easy"}
	if err := run(context.Background(), args, &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"requests/s", "select", "reward", "arm choices"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out.String())
		}
	}
}

func TestRunRejectsBadInput(t *testing.T) {
	for _, args := range [][]string{
		{"-scenario", "nope"},
		{"-c", "0"},
		{"-bogus"},
	} {
		if err := run(context.Background(), args, io.Discard, io.Discard); err == nil {
			t.Errorf("%v: expected an error", args)
		}
	}
	if err := run(context.Background(), []string{"-h"}, io.Discard, io.Discard); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("-h: error = %v, want flag.ErrHelp", err)
	}
}
