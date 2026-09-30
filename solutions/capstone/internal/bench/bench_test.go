package bench

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"

	pb "banditlab/internal/gen/bandit/v1"
	"banditlab/internal/services/aggregator"
	"log/slog"
)

func smallConfig() Config {
	return Config{
		Policies: []string{"thompson"}, Lags: []time.Duration{0},
		Replicas: 2, Workers: 4, Selections: 300, Repeats: 2,
		Probs: []float64{0.2, 0.5, 0.8}, RewardDelay: time.Millisecond, Seed: 1,
	}
}

func TestRunProducesSaneNumbers(t *testing.T) {
	cfg := smallConfig()
	cfg.Policies = []string{"thompson", "ucb1"}
	cfg.Lags = []time.Duration{0, 10 * time.Millisecond}
	points, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 4 {
		t.Fatalf("got %d points, want 4 (2 policies x 2 lags)", len(points))
	}
	for _, p := range points {
		// Expected regret per selection is at most best-worst = 0.6.
		if p.Regret < 0 || p.Regret > 600 || p.BestShare < 0 || p.BestShare > 1 || p.Rate <= 0 {
			t.Errorf("implausible point %+v", p)
		}
	}
	if points[0].Policy != "thompson" || points[2].Policy != "ucb1" || points[1].Lag != 10*time.Millisecond {
		t.Errorf("points are not in (policy, lag) order: %+v", points)
	}
}

func TestRunRejectsBadConfig(t *testing.T) {
	for name, mutate := range map[string]func(*Config){
		"no policies":    func(c *Config) { c.Policies = nil },
		"no replicas":    func(c *Config) { c.Replicas = 0 },
		"negative lag":   func(c *Config) { c.Lags = []time.Duration{-1} },
		"no arms":        func(c *Config) { c.Probs = nil },
		"unknown policy": func(c *Config) { c.Policies = []string{"nope"} },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := smallConfig()
			mutate(&cfg)
			if _, err := Run(context.Background(), cfg); err == nil {
				t.Error("want an error")
			}
		})
	}
}

func TestRunStopsWhenTheContextEnds(t *testing.T) {
	cfg := smallConfig()
	cfg.Selections = 1_000_000
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := Run(ctx, cfg); err == nil {
		t.Fatal("want an error from the cancelled run")
	}
	if took := time.Since(start); took > 5*time.Second {
		t.Errorf("took %v to stop", took)
	}
}

func TestLagDelaysEachSnapshotIndependently(t *testing.T) {
	agg := aggregator.New(2, slog.New(slog.DiscardHandler))
	srv := newInproc(func(s *grpc.Server) { pb.RegisterAggregatorServiceServer(s, agg) })
	defer srv.stop()
	conn, err := srv.dial()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := pb.NewAggregatorServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	first := func(c pb.AggregatorServiceClient) time.Duration {
		start := time.Now()
		w, err := c.Watch(ctx, &pb.WatchRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Recv(); err != nil {
			t.Fatal(err)
		}
		return time.Since(start)
	}
	if fast := first(laggedClient{client, 0}); fast > 100*time.Millisecond {
		t.Errorf("no lag: first snapshot took %v", fast)
	}
	if slow := first(laggedClient{client, 150 * time.Millisecond}); slow < 150*time.Millisecond {
		t.Errorf("150ms lag: first snapshot took %v", slow)
	}

	// Ten changes in quick succession. Delivered one after another with a
	// pause each they would take a second; delayed independently they arrive
	// together, about one lag after they were sent.
	w, _ := laggedClient{client, 100 * time.Millisecond}.Watch(ctx, &pb.WatchRequest{})
	w.Recv() // the initial state
	start := time.Now()
	for range 10 {
		client.Report(ctx, &pb.ReportRequest{Deltas: []*pb.ArmTotals{{Arm: 0, Pulls: 1, RewardSum: 1}, {Arm: 1}}})
	}
	var last uint64
	for last < 10 {
		snap, err := w.Recv()
		if err != nil {
			t.Fatal(err)
		}
		last = snap.GetVersion()
	}
	if took := time.Since(start); took > 500*time.Millisecond {
		t.Errorf("ten snapshots took %v to arrive with a 100ms lag; lag is accumulating", took)
	}
}

// Replicas that hear about rewards late choose worse. The setting is extreme
// on purpose (a lag longer than the whole run), so that the difference is
// large compared with the run-to-run noise of a timing-dependent test.
func TestStalenessCostsRegret(t *testing.T) {
	cfg := smallConfig()
	cfg.Selections = 1500
	cfg.Replicas = 3
	cfg.Workers = 6
	cfg.Repeats = 3
	cfg.Lags = []time.Duration{0, 5 * time.Second}
	points, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	fresh, stale := points[0], points[1]
	if stale.Regret < 1.5*fresh.Regret {
		t.Errorf("regret per 1000: %.1f with fresh snapshots, %.1f with a lag longer than the run; want the stale one clearly worse", fresh.Regret, stale.Regret)
	}
}
