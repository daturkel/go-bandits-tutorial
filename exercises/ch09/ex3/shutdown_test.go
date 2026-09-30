package ex3

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func startServer(t *testing.T, h http.Handler) (*http.Server, string, <-chan error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: h}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ln) }()
	t.Cleanup(func() { srv.Close() })
	return srv, "http://" + ln.Addr().String(), done
}

func TestShutdownIdle(t *testing.T) {
	srv, _, done := startServer(t, http.NotFoundHandler())
	if err := Shutdown(srv, time.Second); err != nil {
		t.Fatalf("Shutdown = %v, want nil", err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Serve returned %v, want http.ErrServerClosed", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the server was still running after Shutdown returned")
	}
}

func TestShutdownLetsRequestsFinish(t *testing.T) {
	arrived := make(chan struct{})
	srv, url, _ := startServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(arrived)
		time.Sleep(200 * time.Millisecond)
		io.WriteString(w, "done")
	}))
	body := make(chan string, 1)
	go func() {
		resp, err := http.Get(url)
		if err != nil {
			body <- "error: " + err.Error()
			return
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		body <- string(b)
	}()
	<-arrived

	if err := Shutdown(srv, 5*time.Second); err != nil {
		t.Fatalf("Shutdown = %v, want nil", err)
	}
	if got := <-body; got != "done" {
		t.Errorf("client got %q, want the finished response", got)
	}
}

func TestShutdownGraceExpires(t *testing.T) {
	arrived := make(chan struct{})
	release := make(chan struct{})
	srv, url, _ := startServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(arrived)
		<-release
	}))
	t.Cleanup(func() { close(release) })
	result := make(chan error, 1)
	go func() {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
		}
		result <- err
	}()
	<-arrived

	start := time.Now()
	err := Shutdown(srv, 100*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown = %v, want an error wrapping context.DeadlineExceeded", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Error("Shutdown waited far longer than the grace period")
	}
	select {
	case err := <-result:
		if err == nil {
			t.Error("the stuck request should have been cut off")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the stuck request's connection was never closed")
	}
}
