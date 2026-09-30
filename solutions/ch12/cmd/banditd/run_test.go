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
	"banditlab/internal/store/storetest"
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
		env("BANDIT_ADDR", "127.0.0.1:0", "BANDIT_LOG_FORMAT", "json", "BANDIT_POLICY", "ucb1", "BANDIT_LOG_LEVEL", "debug"))

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

func stat(t *testing.T, addr string) (pulls int) {
	t.Helper()
	resp, err := http.Get("http://" + addr + "/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		Arms []struct{ Pulls int } `json:"arms"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	for _, a := range body.Arms {
		pulls += a.Pulls
	}
	return pulls
}

// The whole daemon against a real PostgreSQL: run it, feed it, stop it, run
// it again on the same database, and it remembers.
func TestRunKeepsStateAcrossRestartsInPostgres(t *testing.T) {
	url := storetest.DatabaseURL(t)
	schema := storetest.NewSchema(t, url)
	dbURL := url + "&search_path=" + schema // libpq-style runtime parameter
	environ := env("BANDIT_ADDR", "127.0.0.1:0", "BANDIT_DATABASE_URL", dbURL, "BANDIT_LOG_FORMAT", "json")

	addr, _, stop := start(t, environ)
	for range 5 {
		resp, err := http.Post("http://"+addr+"/select", "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		var sel struct {
			RequestID string `json:"request_id"`
		}
		json.NewDecoder(resp.Body).Decode(&sel)
		resp.Body.Close()
		body := strings.NewReader(`{"request_id":"` + sel.RequestID + `","reward":1}`)
		resp, err = http.Post("http://"+addr+"/reward", "application/json", body)
		if err != nil || resp.StatusCode != http.StatusNoContent {
			t.Fatalf("reward: %v %v", resp, err)
		}
		resp.Body.Close()
	}
	if got := stat(t, addr); got != 5 {
		t.Fatalf("first run counts %d pulls, want 5", got)
	}
	if err := stop(); err != nil {
		t.Fatal(err)
	}

	addr, logs, stop := start(t, environ)
	defer stop()
	if got := stat(t, addr); got != 5 {
		t.Errorf("after restart the service counts %d pulls, want the 5 it had", got)
	}
	if !strings.Contains(logs.String(), "database connected") {
		t.Errorf("startup did not report the database: %s", logs.String())
	}
}

func TestRunWithoutDatabaseWarns(t *testing.T) {
	_, logs, stop := start(t, env("BANDIT_ADDR", "127.0.0.1:0"))
	defer stop()
	if !strings.Contains(logs.String(), "state is kept in memory") {
		t.Errorf("no warning about volatile state:\n%s", logs.String())
	}
}

func TestRunBadDatabaseURL(t *testing.T) {
	err := run(context.Background(), nil, env("BANDIT_ADDR", "127.0.0.1:0", "BANDIT_DATABASE_URL", "postgres://nobody@127.0.0.1:1/none?connect_timeout=1"), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "open database") {
		t.Errorf("error = %v, want one about opening the database", err)
	}
}
