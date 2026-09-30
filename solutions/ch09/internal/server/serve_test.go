package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

// blockingServer serves a handler that holds every request open until
// release is closed, and reports when a request has arrived.
type blockingServer struct {
	url      string
	arrived  chan struct{}
	release  chan struct{}
	done     chan error
	cancel   context.CancelFunc
	shutdown atomic.Bool
}

func startBlocking(t *testing.T, grace time.Duration) *blockingServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	b := &blockingServer{
		url:     "http://" + ln.Addr().String(),
		arrived: make(chan struct{}, 1),
		release: make(chan struct{}),
		done:    make(chan error, 1),
	}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.arrived <- struct{}{}
		<-b.release
		w.Write([]byte("finished"))
	})
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	go func() {
		b.done <- Serve(ctx, ln, h, log, ServeOptions{
			ShutdownGrace: grace,
			OnShutdown:    func() { b.shutdown.Store(true) },
		})
	}()
	t.Cleanup(func() {
		select {
		case <-b.release:
		default:
			close(b.release)
		}
		cancel()
	})
	return b
}

// get starts a request in the background and reports its outcome.
func get(url string) <-chan string {
	out := make(chan string, 1)
	go func() {
		resp, err := http.Get(url)
		if err != nil {
			out <- "error"
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		out <- string(body)
	}()
	return out
}

func TestServeDrainsInFlightRequests(t *testing.T) {
	b := startBlocking(t, 5*time.Second)
	result := get(b.url)
	<-b.arrived // the request is now inside the handler

	b.cancel() // begin shutdown
	select {
	case err := <-b.done:
		t.Fatalf("Serve returned (%v) while a request was still running", err)
	case <-time.After(100 * time.Millisecond):
	}
	if !b.shutdown.Load() {
		t.Error("OnShutdown was not called when shutdown began")
	}

	close(b.release) // let the request finish
	if got := <-result; got != "finished" {
		t.Errorf("client got %q, want the completed response", got)
	}
	select {
	case err := <-b.done:
		if err != nil {
			t.Errorf("Serve = %v after a clean drain, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return after the last request finished")
	}
}

func TestServeGraceExpires(t *testing.T) {
	b := startBlocking(t, 100*time.Millisecond)
	result := get(b.url)
	<-b.arrived

	b.cancel()
	select {
	case err := <-b.done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Serve = %v, want one wrapping context.DeadlineExceeded", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not give up after the grace period")
	}
	if got := <-result; got != "error" {
		t.Errorf("client got %q; its connection should have been cut", got)
	}
}

// During the drain delay the server still accepts connections but reports
// itself unhealthy; afterwards it stops listening.
func TestServeDrainDelay(t *testing.T) {
	s := New(newPolicy(t), quietLogger())
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, ln, s.Handler(), quietLogger(), ServeOptions{
			ShutdownGrace: time.Second,
			OnShutdown:    s.SetDraining,
			DrainDelay:    300 * time.Millisecond,
		})
	}()
	url := "http://" + ln.Addr().String() + "/healthz"

	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("before shutdown: %v, %v", resp, err)
	}
	resp.Body.Close()

	cancel()
	time.Sleep(100 * time.Millisecond) // inside the drain delay
	resp, err = http.Get(url)
	if err != nil {
		t.Fatalf("the listener closed during the drain delay: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("health during drain = %d, want 503", resp.StatusCode)
	}

	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if _, err := http.Get(url); err == nil {
		t.Error("server still accepting connections after shutdown")
	}
}

func TestServeIdleShutdownIsImmediate(t *testing.T) {
	b := startBlocking(t, 5*time.Second)
	start := time.Now()
	b.cancel()
	if err := <-b.done; err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > time.Second {
		t.Errorf("idle shutdown took %v", time.Since(start))
	}
}
