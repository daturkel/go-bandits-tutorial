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
	pol := daemontest.Start(t, policyd.Run, daemontest.NoEnv, "-addr", "127.0.0.1:0", "-policy", "ucb1", "-instance", "p1")
	fb := daemontest.Start(t, feedbackd.Run, daemontest.NoEnv, "-addr", "127.0.0.1:0")

	out, err := call(t, "", "-addr", pol.Addr, "health")
	if err != nil || strings.TrimSpace(out) != "SERVING" {
		t.Errorf("health = %q, %v", out, err)
	}
	out, err = call(t, "", "-addr", pol.Addr, "select")
	if err != nil || !strings.Contains(out, "request_id=") || !strings.Contains(out, "arm=0") || !strings.Contains(out, "instance=p1") {
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

// Two replicas behind one target: with round_robin, calls alternate between
// them; with the default they all go to the first.
func TestRoundRobinSpreadsCalls(t *testing.T) {
	a := daemontest.Start(t, policyd.Run, daemontest.NoEnv, "-addr", "127.0.0.1:0", "-instance", "a")
	b := daemontest.Start(t, policyd.Run, daemontest.NoEnv, "-addr", "127.0.0.1:0", "-instance", "b")
	target := a.Addr + "," + b.Addr

	count := func(args ...string) map[string]int {
		out, err := call(t, "", args...)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]int{}
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			_, inst, _ := strings.Cut(line, "instance=")
			seen[inst]++
		}
		return seen
	}
	// The first calls can all reach the replica that connected first, so
	// count over enough calls that the split is clear.
	if got := count("-addr", target, "-lb", "round_robin", "select", "40"); got["a"] < 12 || got["b"] < 12 {
		t.Errorf("round_robin: %v, want both replicas well used", got)
	}
	if got := count("-addr", target, "select", "40"); len(got) != 1 {
		t.Errorf("pick_first: %v, want a single instance", got)
	}
}

func TestHealthFailsWhenNotServing(t *testing.T) {
	// Nothing is listening on this port.
	if _, err := call(t, "", "-addr", "127.0.0.1:1", "-timeout", "300ms", "health"); err == nil {
		t.Error("health of a missing service should fail")
	}
}
