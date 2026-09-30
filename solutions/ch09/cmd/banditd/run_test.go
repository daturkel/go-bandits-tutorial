package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"banditlab/internal/config"
)

// syncBuffer lets the test read logs while the server is still writing them.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// addrRE finds the listen address in either log format (addr=... or "addr":"...").
var addrRE = regexp.MustCompile(`"?addr"?[=:]"?([0-9.:]+)`)

func env(pairs ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return func(k string) string { return m[k] }
}

// start runs banditd on a free port and waits until it is listening.
func start(t *testing.T, getenv func(string) string, args ...string) (addr string, logs *syncBuffer, stop func() error) {
	t.Helper()
	logs = &syncBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run(ctx, args, getenv, logs) }()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if m := addrRE.FindStringSubmatch(logs.String()); m != nil {
			addr = m[1]
			break
		}
		select {
		case err := <-done:
			t.Fatalf("run exited early: %v\nlogs: %s", err, logs.String())
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not start; logs: %s", logs.String())
		}
		time.Sleep(5 * time.Millisecond)
	}
	stop = func() error {
		cancel()
		select {
		case err := <-done:
			return err
		case <-time.After(3 * time.Second):
			t.Fatal("run did not return after cancellation")
			return nil
		}
	}
	t.Cleanup(func() { cancel() })
	return addr, logs, stop
}

func TestRunServesWithJSONLogsAndStopsCleanly(t *testing.T) {
	addr, logs, stop := start(t,
		env("BANDIT_ADDR", "127.0.0.1:0", "BANDIT_LOG_FORMAT", "json", "BANDIT_POLICY", "ucb1"))

	resp, err := http.Post("http://"+addr+"/select", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("select status = %d", resp.StatusCode)
	}

	if err := stop(); err != nil {
		t.Fatalf("run returned %v after a clean stop, want nil", err)
	}

	sawRequest, sawComplete := false, false
	for _, line := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log line %q is not JSON: %v", line, err)
		}
		if rec["service"] != "banditd" {
			t.Errorf("record lacks the service attribute: %v", rec)
		}
		sawRequest = sawRequest || (rec["msg"] == "request" && rec["path"] == "/select")
		sawComplete = sawComplete || rec["msg"] == "shutdown complete"
	}
	if !sawRequest || !sawComplete {
		t.Errorf("missing log records (request %v, shutdown complete %v):\n%s", sawRequest, sawComplete, logs.String())
	}
}

func TestRunFlagsOverrideEnvironment(t *testing.T) {
	addr, logs, stop := start(t,
		env("BANDIT_ADDR", "127.0.0.1:1", "BANDIT_POLICY", "ucb1"), "-addr", "127.0.0.1:0", "-policy", "thompson")
	defer stop()
	if !strings.Contains(logs.String(), "thompson") || addr == "127.0.0.1:1" {
		t.Errorf("flags did not take precedence: addr %s\n%s", addr, logs.String())
	}
}

func TestRunRejectsBadConfiguration(t *testing.T) {
	for name, tc := range map[string]struct {
		env  func(string) string
		args []string
	}{
		"bad env":     {env("BANDIT_ARMS", "many"), nil},
		"bad policy":  {env(), []string{"-policy", "nope"}},
		"zero arms":   {env(), []string{"-arms", "0"}},
		"bad logging": {env(), []string{"-log-format", "xml"}},
	} {
		t.Run(name, func(t *testing.T) {
			err := run(context.Background(), tc.args, tc.env, &bytes.Buffer{})
			if !errors.Is(err, config.ErrInvalid) {
				t.Errorf("error = %v, want config.ErrInvalid", err)
			}
		})
	}
}

func TestRunPortInUse(t *testing.T) {
	addr, _, stop := start(t, env("BANDIT_ADDR", "127.0.0.1:0"))
	defer stop()
	err := run(context.Background(), []string{"-addr", addr}, env(), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected an error when the port is taken")
	}
	if errors.Is(err, config.ErrInvalid) {
		t.Errorf("a busy port is an environment problem, not invalid config: %v", err)
	}
}
