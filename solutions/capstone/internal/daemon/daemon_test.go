package daemon_test

import (
	"context"
	"errors"
	"flag"
	"io"
	"testing"
	"time"

	"banditlab/internal/apps/aggregatord"
	"banditlab/internal/apps/feedbackd"
	"banditlab/internal/apps/policyd"
	"banditlab/internal/daemon"
)

func env(pairs ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return func(k string) string { return m[k] }
}

func TestCommonDefaultsAndPrecedence(t *testing.T) {
	load := func(getenv func(string) string, args ...string) (daemon.Common, error) {
		var c daemon.Common
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		e := daemon.NewEnv(getenv)
		finish := c.Register(fs, e, "localhost:1234")
		if err := fs.Parse(args); err != nil {
			return c, err
		}
		if err := finish(); err != nil {
			return c, err
		}
		return c, e.Err()
	}

	c, err := load(env())
	if err != nil || c.Addr != "localhost:1234" || c.Grace != 10*time.Second || c.LogFormat != "text" {
		t.Errorf("defaults = %+v, %v", c, err)
	}
	c, _ = load(env("BANDIT_ADDR", ":7000"))
	if c.Addr != ":7000" {
		t.Errorf("env: Addr = %q, want :7000", c.Addr)
	}
	c, _ = load(env("BANDIT_ADDR", ":7000"), "-addr", ":8000")
	if c.Addr != ":8000" {
		t.Errorf("flag over env: Addr = %q, want :8000", c.Addr)
	}
}

func TestBadSettingsAreReportedTogether(t *testing.T) {
	_, err := func() (daemon.Common, error) {
		var c daemon.Common
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		e := daemon.NewEnv(env("BANDIT_SHUTDOWN_GRACE", "soon", "BANDIT_LOG_LEVEL", "loud"))
		finish := c.Register(fs, e, "x")
		fs.Parse(nil)
		if err := e.Err(); err != nil {
			return c, err
		}
		return c, finish()
	}()
	if !errors.Is(err, daemon.ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid", err)
	}
}

func TestEveryDaemonRejectsBadSettingsBeforeListening(t *testing.T) {
	runs := map[string]func(context.Context, []string, func(string) string, io.Writer) error{
		"policyd": policyd.Run, "feedbackd": feedbackd.Run, "aggregatord": aggregatord.Run,
	}
	for name, run := range runs {
		for _, args := range [][]string{{"-arms", "0"}, {"-log-format", "xml"}, {"-shutdown-grace", "-1s"}} {
			err := run(context.Background(), args, env(), io.Discard)
			if !errors.Is(err, daemon.ErrInvalid) {
				t.Errorf("%s %v: err = %v, want ErrInvalid", name, args, err)
			}
		}
		if err := run(context.Background(), []string{"-h"}, env(), io.Discard); !errors.Is(err, flag.ErrHelp) {
			t.Errorf("%s -h: err = %v, want flag.ErrHelp", name, err)
		}
	}
	if err := policyd.Run(context.Background(), []string{"-policy", "nope"}, env(), io.Discard); !errors.Is(err, daemon.ErrInvalid) {
		t.Errorf("unknown policy: err = %v, want ErrInvalid", err)
	}
}
