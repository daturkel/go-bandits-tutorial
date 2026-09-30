package ex3

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMaxInFlight(t *testing.T) {
	const limit = 3
	var running atomic.Int32
	release := make(chan struct{})
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		running.Add(1)
		defer running.Add(-1)
		select {
		case <-release:
		case <-time.After(5 * time.Second):
		}
		w.WriteHeader(http.StatusOK)
	})
	ts := httptest.NewServer(MaxInFlight(limit)(slow))
	defer ts.Close()
	client := &http.Client{Timeout: 2 * time.Second}

	// Fill every slot.
	var wg sync.WaitGroup
	codes := make(chan int, limit)
	for range limit {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(ts.URL)
			if err != nil {
				codes <- -1
				return
			}
			resp.Body.Close()
			codes <- resp.StatusCode
		}()
	}
	for deadline := time.Now().Add(2 * time.Second); running.Load() < limit; {
		if time.Now().After(deadline) {
			t.Fatalf("only %d requests reached the handler", running.Load())
		}
		time.Sleep(time.Millisecond)
	}

	// One more must be turned away at once.
	start := time.Now()
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("the extra request got no response (was it queued instead of rejected?): %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("extra request status = %d, want 503", resp.StatusCode)
	}
	if resp.Header.Get("Retry-After") != "1" {
		t.Errorf("Retry-After = %q, want 1", resp.Header.Get("Retry-After"))
	}
	if time.Since(start) > time.Second {
		t.Error("the rejection was not immediate")
	}

	// The admitted requests still complete, and their slots free up.
	close(release)
	wg.Wait()
	close(codes)
	for code := range codes {
		if code != http.StatusOK {
			t.Errorf("admitted request finished with %d", code)
		}
	}
	resp, err = client.Get(ts.URL)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Errorf("after the others finished: (%v, %v), want 200", resp, err)
	}
}
