// Package daemontest starts a daemon's Run function inside a test and waits
// until it is listening.
package daemontest

import (
	"bytes"
	"context"
	"io"
	"regexp"
	"sync"
	"testing"
	"time"
)

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

// Daemon is a running daemon.
type Daemon struct {
	Addr string // where it listens
	Logs func() string
	stop func() error
}

// Stop cancels the daemon's context and returns what Run returned.
func (d *Daemon) Stop() error { return d.stop() }

// Start runs run with args on a free port (pass "-addr", "127.0.0.1:0") and
// returns once it has logged its address. The daemon is stopped when the test
// ends, if the test has not stopped it already.
func Start(t testing.TB, run func(context.Context, []string, func(string) string, io.Writer) error, getenv func(string) string, args ...string) *Daemon {
	t.Helper()
	logs := &syncBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run(ctx, args, getenv, logs) }()

	var once sync.Once
	var result error
	stop := func() error {
		once.Do(func() {
			cancel()
			select {
			case result = <-done:
			case <-time.After(10 * time.Second):
				t.Errorf("daemon did not stop within 10s; logs:\n%s", logs.String())
			}
		})
		return result
	}
	t.Cleanup(func() { stop() })

	deadline := time.Now().Add(5 * time.Second)
	for {
		if m := addrRE.FindStringSubmatch(logs.String()); m != nil {
			return &Daemon{Addr: m[1], Logs: logs.String, stop: stop}
		}
		select {
		case err := <-done:
			t.Fatalf("daemon exited early: %v\nlogs: %s", err, logs.String())
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("daemon did not start; logs:\n%s", logs.String())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// NoEnv is an empty environment.
func NoEnv(string) string { return "" }
