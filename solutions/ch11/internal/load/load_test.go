package load

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"banditlab/internal/bandit"
	"banditlab/internal/server"
	"banditlab/internal/store"
)

// startServer runs a real banditd handler and returns its URL and stats.
func startServer(t *testing.T, opts ...server.Option) (url string, stats func() (selects, updates int64, pulls int)) {
	t.Helper()
	pol, err := bandit.NewSnapshotPolicy("thompson", 3, bandit.NewRNG(1, bandit.StreamPolicy))
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ts := httptest.NewServer(server.New(pol, store.NewMemory(3), log, opts...).Handler())
	t.Cleanup(ts.Close)
	return ts.URL, func() (int64, int64, int) {
		resp, err := http.Get(ts.URL + "/stats")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var s struct {
			Selects, Updates int64
			Arms             []struct{ Pulls int }
		}
		json.NewDecoder(resp.Body).Decode(&s)
		pulls := 0
		for _, a := range s.Arms {
			pulls += a.Pulls
		}
		return s.Selects, s.Updates, pulls
	}
}

func TestClosedLoopCountsMatchTheServer(t *testing.T) {
	url, stats := startServer(t)
	rep, err := Run(context.Background(), Config{
		BaseURL: url, Workers: 4, Elapsed: 300 * time.Millisecond,
		Probs: []float64{0.2, 0.5, 0.8}, Seed: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Select.Count == 0 || rep.Select.Errors != 0 || rep.Reward.Errors != 0 {
		t.Fatalf("report = %+v", rep)
	}
	// A call still in flight when the run ends may be counted by the server
	// and not the client (the client discards it), never the other way
	// round, and there can be at most one such call per worker.
	selects, updates, pulls := stats()
	if int(selects) < rep.Select.Count || int(selects)-rep.Select.Count > 4 ||
		int(updates) < rep.Reward.Count || int(updates)-rep.Reward.Count > 4 || pulls != int(updates) {
		t.Errorf("client saw %d selects and %d rewards; server saw %d selects, %d updates, %d pulls",
			rep.Select.Count, rep.Reward.Count, selects, updates, pulls)
	}
	total := 0
	for _, n := range rep.Pulls {
		total += n
	}
	if total != rep.Select.Count {
		t.Errorf("arm choices sum to %d, want %d", total, rep.Select.Count)
	}
	if !(rep.Select.P50 <= rep.Select.P90 && rep.Select.P90 <= rep.Select.P99 && rep.Select.P99 <= rep.Select.Max) {
		t.Errorf("percentiles are out of order: %+v", rep.Select)
	}
	if rep.Status[200] == 0 || rep.Status[204] == 0 {
		t.Errorf("status counts = %v, want 200s and 204s", rep.Status)
	}
}

func TestRewardFractionLeavesSelectionsUnrewarded(t *testing.T) {
	url, stats := startServer(t)
	rep, err := Run(context.Background(), Config{
		BaseURL: url, Workers: 4, Elapsed: 300 * time.Millisecond,
		Probs: []float64{0.2, 0.5, 0.8}, Seed: 1, RewardFraction: 0.25,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Select.Count < 40 {
		t.Fatalf("only %d selections; the run is too short to judge a fraction", rep.Select.Count)
	}
	share := float64(rep.Reward.Count) / float64(rep.Select.Count)
	if share < 0.1 || share > 0.4 {
		t.Errorf("%.2f of selections were rewarded, want about 0.25", share)
	}
	if _, updates, _ := stats(); int(updates) < rep.Reward.Count || int(updates)-rep.Reward.Count > 4 {
		t.Errorf("server recorded %d rewards, client counted %d", updates, rep.Reward.Count)
	}
}

func TestOpenLoopArrivalRate(t *testing.T) {
	url, _ := startServer(t)
	rep, err := Run(context.Background(), Config{
		BaseURL: url, Workers: 4, Elapsed: time.Second, Rate: 100,
		Probs: []float64{0.2, 0.5, 0.8}, Seed: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Select.Count < 80 || rep.Select.Count > 110 {
		t.Errorf("%d selections in 1s at 100/s", rep.Select.Count)
	}
	if rep.Dropped != 0 {
		t.Errorf("%d arrivals dropped at a rate the server easily handles", rep.Dropped)
	}
}

// When the server is slower than the arrival rate, an open-loop test shows
// the queueing in its latencies. A closed-loop test would just send fewer
// requests and look fine (the "coordinated omission" problem).
func TestOpenLoopExposesQueueing(t *testing.T) {
	url, _ := startServer(t, server.WithSelectDelay(50*time.Millisecond))
	rep, err := Run(context.Background(), Config{
		BaseURL: url, Workers: 2, Elapsed: time.Second, Rate: 200, // capacity is about 2/0.05 = 40/s
		Probs: []float64{0.2, 0.5, 0.8}, Seed: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Select.P99 < 300*time.Millisecond {
		t.Errorf("p99 = %v: waiting in the backlog should have made it far larger than the 50ms service time", rep.Select.P99)
	}
	if rep.Select.Count > 100 {
		t.Errorf("%d selections completed in 1s with capacity of about 40/s", rep.Select.Count)
	}
}

func TestRunValidatesConfig(t *testing.T) {
	if _, err := Run(context.Background(), Config{BaseURL: "http://x", Workers: 0, Probs: []float64{0.5}}); err == nil {
		t.Error("zero workers accepted")
	}
	if _, err := Run(context.Background(), Config{BaseURL: "http://x", Workers: 1}); err == nil {
		t.Error("no arms accepted")
	}
}

func TestServerErrorsAreCounted(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer ts.Close()
	rep, err := Run(context.Background(), Config{
		BaseURL: ts.URL, Workers: 2, Elapsed: 100 * time.Millisecond, Probs: []float64{0.5, 0.5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Select.Count == 0 || rep.Select.Errors != rep.Select.Count || rep.Status[503] != rep.Select.Count {
		t.Errorf("report = %+v", rep)
	}
}

func TestReportText(t *testing.T) {
	url, _ := startServer(t)
	rep, err := Run(context.Background(), Config{BaseURL: url, Workers: 2, Elapsed: 100 * time.Millisecond, Probs: []float64{0.3, 0.6, 0.5}})
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	rep.Write(&sb)
	for _, want := range []string{"requests/s", "select", "reward", "p99", "arm choices", "expected regret"} {
		if !strings.Contains(sb.String(), want) {
			t.Errorf("report lacks %q:\n%s", want, sb.String())
		}
	}
}

// A scenario with fewer arms than the service must be reported, not crash.
func TestArmCountMismatchIsAnError(t *testing.T) {
	url, _ := startServer(t) // three arms
	rep, err := Run(context.Background(), Config{
		BaseURL: url, Workers: 1, Elapsed: 200 * time.Millisecond, Probs: []float64{0.5}, // one arm
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Select.Errors == 0 || rep.Status[-1] == 0 {
		t.Errorf("mismatch went unnoticed: %+v", rep)
	}
}
