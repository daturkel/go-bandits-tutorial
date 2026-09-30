package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"banditlab/bandit"
	"banditlab/server"
)

var update = flag.Bool("update", false, "rewrite golden files")

// TestGolden pins the exact output of a seeded run. If a change alters the
// numbers, the diff shows up in review; run with -update to accept it.
func TestGolden(t *testing.T) {
	var buf bytes.Buffer
	if err := run(context.Background(), []string{"run", "-scenario", "easy", "-steps", "5000", "-seed", "42"}, &buf, io.Discard); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("testdata", "easy_seed42.golden")
	if *update {
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("output differs from %s:\n--- got ---\n%s--- want ---\n%s", path, buf.Bytes(), want)
	}
}

func TestRunUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no command", nil},
		{"unknown command", []string{"walk"}},
		{"unknown scenario", []string{"run", "-scenario", "nope"}},
		{"unknown policy", []string{"run", "-policy", "nope"}},
		{"bad flag", []string{"run", "-bogus"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := run(context.Background(), tc.args, &bytes.Buffer{}, io.Discard)
			if !errors.Is(err, errUsage) {
				t.Fatalf("error = %v, want a usage error", err)
			}
		})
	}
}

func TestCompareWritesFiles(t *testing.T) {
	dir := t.TempDir()
	csvPath, svgPath := filepath.Join(dir, "r.csv"), filepath.Join(dir, "r.svg")
	var buf bytes.Buffer
	args := []string{"compare", "-scenario", "easy", "-steps", "300", "-seeds", "4", "-csv", csvPath, "-svg", svgPath}
	if err := run(context.Background(), args, &buf, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{csvPath, svgPath} {
		if info, err := os.Stat(p); err != nil || info.Size() == 0 {
			t.Errorf("%s not written: %v", p, err)
		}
	}
	if !bytes.Contains(buf.Bytes(), []byte("thompson")) {
		t.Errorf("summary missing thompson:\n%s", buf.String())
	}
}

func TestCompareTimeoutFlag(t *testing.T) {
	args := []string{"compare", "-seeds", "5000", "-steps", "200000", "-timeout", "30ms"}
	err := run(context.Background(), args, io.Discard, io.Discard)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
}

func TestCompareWorkersFlagGivesSameOutput(t *testing.T) {
	outputs := map[string]string{}
	for _, w := range []string{"1", "4"} {
		var buf bytes.Buffer
		args := []string{"compare", "-scenario", "easy", "-steps", "500", "-seeds", "10", "-workers", w}
		if err := run(context.Background(), args, &buf, io.Discard); err != nil {
			t.Fatal(err)
		}
		outputs[w] = buf.String()
	}
	if outputs["1"] != outputs["4"] {
		t.Errorf("workers=1 and workers=4 printed different tables:\n%s\n%s", outputs["1"], outputs["4"])
	}
}

// syncBuffer lets the test read what the progress goroutine wrote.
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

func TestReportProgress(t *testing.T) {
	var out syncBuffer
	update, stop := reportProgress(context.Background(), &out, 5*time.Millisecond)
	update(3, 10)
	time.Sleep(60 * time.Millisecond)
	stop()
	if got := out.String(); !strings.Contains(got, "3/10 runs") || !strings.HasSuffix(got, "\n") {
		t.Errorf("progress output %q should report 3/10 and end with a newline", got)
	}
}

func TestReportProgressStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, stop := reportProgress(ctx, io.Discard, time.Hour)
	cancel()
	done := make(chan struct{})
	go func() {
		stop() // must return even though the ticker never fires
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("stop did not return after the context was cancelled")
	}
}

func TestServeAndStop(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	pol, err := bandit.NewSnapshotPolicy("thompson", 3, bandit.NewRNG(1, bandit.StreamPolicy))
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, ln, server.New(pol, log).Handler(), log) }()

	resp, err := http.Get("http://" + ln.Addr().String() + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz status = %d", resp.StatusCode)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serve returned %v after cancellation, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not return after the context was cancelled")
	}
}

func TestServeRejectsBadPolicy(t *testing.T) {
	err := run(context.Background(), []string{"serve", "-policy", "nope"}, io.Discard, io.Discard)
	if !errors.Is(err, errUsage) {
		t.Errorf("error = %v, want a usage error", err)
	}
}

func TestServePortInUse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	err = run(context.Background(), []string{"serve", "-addr", ln.Addr().String()}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("expected an error when the port is taken")
	}
}
