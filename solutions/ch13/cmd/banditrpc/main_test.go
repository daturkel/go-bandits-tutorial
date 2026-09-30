package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"banditlab/internal/apps/feedbackd"
	"banditlab/internal/apps/policyd"
	"banditlab/internal/daemon/daemontest"
)

func call(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := run(context.Background(), args, strings.NewReader(stdin), &out, &out)
	return out.String(), err
}

func TestAgainstRealDaemons(t *testing.T) {
	pol := daemontest.Start(t, policyd.Run, daemontest.NoEnv, "-addr", "127.0.0.1:0", "-policy", "ucb1")
	fb := daemontest.Start(t, feedbackd.Run, daemontest.NoEnv, "-addr", "127.0.0.1:0")

	out, err := call(t, "", "-addr", pol.Addr, "health")
	if err != nil || strings.TrimSpace(out) != "SERVING" {
		t.Errorf("health = %q, %v", out, err)
	}
	out, err = call(t, "", "-addr", pol.Addr, "select")
	if err != nil || !strings.Contains(out, "request_id=") || !strings.Contains(out, "arm=0") {
		t.Errorf("select = %q, %v", out, err)
	}
	out, err = call(t, "", "-addr", pol.Addr, "stats")
	if err != nil || !strings.Contains(out, "policy=ucb1") || !strings.Contains(out, "pending=1") {
		t.Errorf("stats = %q, %v", out, err)
	}

	// The two daemons have separate in-memory stores here, so the feedback
	// service has never heard of that id, and says so with a status code.
	_, err = call(t, "", "-addr", fb.Addr, "reward", "no-such-id", "1")
	if err == nil || !strings.HasPrefix(err.Error(), "NotFound:") {
		t.Errorf("reward for an unknown id: err = %v, want NotFound", err)
	}
	_, err = call(t, "", "-addr", fb.Addr, "reward", "x", "9")
	if err == nil || !strings.HasPrefix(err.Error(), "InvalidArgument:") {
		t.Errorf("reward of 9: err = %v, want InvalidArgument", err)
	}
	out, err = call(t, "a 1\nb 0\n", "-addr", fb.Addr, "batch")
	if err != nil || strings.TrimSpace(out) != "accepted=0 rejected=2" {
		t.Errorf("batch = %q, %v", out, err)
	}
}

func TestUsageErrors(t *testing.T) {
	if _, err := call(t, ""); err == nil {
		t.Error("no command should be an error")
	}
	if _, err := call(t, "", "frobnicate"); err == nil {
		t.Error("unknown command should be an error")
	}
	if _, err := call(t, "", "reward", "only-one-arg"); err == nil {
		t.Error("reward with one argument should be an error")
	}
}
