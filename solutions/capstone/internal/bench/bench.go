// Package bench runs the whole system, in one process, under load: real
// policy, feedback and aggregator services connected by gRPC streams, driven
// by simulated users, with a controllable delay on the path that carries the
// learned counts back to the policy replicas.
package bench

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"

	"banditlab/internal/bandit"
	pb "banditlab/internal/gen/bandit/v1"
	"banditlab/internal/services/aggregator"
	"banditlab/internal/services/feedback"
	"banditlab/internal/services/policy"
	"banditlab/internal/stat"
	"banditlab/internal/store"
)

// Config describes one comparison.
type Config struct {
	Policies   []string        // policy specs, as for bandit.NewPolicy
	Lags       []time.Duration // snapshot lags to try; 0 is as fast as the system can go
	Replicas   int
	Workers    int // concurrent simulated users
	Selections int // total selections per run
	Repeats    int // independent runs per setting

	Probs       []float64     // the world: success probability per arm
	RewardDelay time.Duration // how long a user takes to react before the reward is sent
	Think       time.Duration // pause between one user and the next
	Seed        uint64
}

// Point is the outcome for one policy at one lag.
type Point struct {
	Policy    string
	Lag       time.Duration
	Regret    float64 // expected regret per 1000 selections, mean over the repeats
	StdErr    float64
	BestShare float64 // fraction of selections that chose the best arm
	Rate      float64 // selections per second
}

func (c Config) validate() error {
	switch {
	case len(c.Policies) == 0 || len(c.Lags) == 0:
		return errors.New("bench: need at least one policy and one lag")
	case c.Replicas < 1 || c.Workers < 1 || c.Selections < 1 || c.Repeats < 1:
		return errors.New("bench: replicas, workers, selections and repeats must be positive")
	case len(c.Probs) == 0:
		return errors.New("bench: no arms")
	case c.RewardDelay < 0 || c.Think < 0:
		return errors.New("bench: delays must not be negative")
	}
	for _, l := range c.Lags {
		if l < 0 {
			return fmt.Errorf("bench: negative lag %v", l)
		}
	}
	return nil
}

// Run measures every policy at every lag. Runs happen one at a time, because
// they compete for the machine and would disturb each other's timing.
func Run(ctx context.Context, cfg Config) ([]Point, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	var points []Point
	for _, spec := range cfg.Policies {
		for _, lag := range cfg.Lags {
			var regrets, shares, rates []float64
			for rep := range cfg.Repeats {
				res, err := runOnce(ctx, cfg, spec, lag, cfg.Seed+uint64(rep))
				if err != nil {
					return nil, fmt.Errorf("policy %q, lag %v, repeat %d: %w", spec, lag, rep, err)
				}
				regrets = append(regrets, res.regret/float64(cfg.Selections)*1000)
				shares = append(shares, res.bestShare)
				rates = append(rates, res.rate)
			}
			points = append(points, Point{
				Policy: spec, Lag: lag,
				Regret: stat.Mean(regrets), StdErr: stat.StdErr(regrets),
				BestShare: stat.Mean(shares), Rate: stat.Mean(rates),
			})
		}
	}
	return points, nil
}

type runResult struct {
	regret    float64
	bestShare float64
	rate      float64
}

// runOnce builds the system, plays cfg.Selections users against it, and tears
// it down.
func runOnce(ctx context.Context, cfg Config, spec string, lag time.Duration, seed uint64) (runResult, error) {
	log := slog.New(slog.DiscardHandler)
	arms := len(cfg.Probs)
	st := store.NewMemory(arms)

	agg := aggregator.New(arms, log)
	aggSrv := newInproc(func(s *grpc.Server) { pb.RegisterAggregatorServiceServer(s, agg) })
	defer aggSrv.stop()
	reportConn, err := aggSrv.dial()
	if err != nil {
		return runResult{}, err
	}
	defer reportConn.Close()

	fb := feedback.New(st, arms, feedback.NewReporter(pb.NewAggregatorServiceClient(reportConn), "bench"), log)
	fbSrv := newInproc(func(s *grpc.Server) { pb.RegisterFeedbackServiceServer(s, fb) })
	defer fbSrv.stop()
	fbConn, err := fbSrv.dial()
	if err != nil {
		return runResult{}, err
	}
	defer fbConn.Close()
	feedbackClient := pb.NewFeedbackServiceClient(fbConn)

	followCtx, stopFollowing := context.WithCancel(ctx)
	var followers sync.WaitGroup
	defer func() {
		stopFollowing()
		followers.Wait()
	}()

	replicas := make([]pb.PolicyServiceClient, cfg.Replicas)
	for r := range replicas {
		pol, err := bandit.NewSnapshotPolicy(spec, arms, bandit.NewRNG(seed, bandit.StreamPolicy+uint64(r)))
		if err != nil {
			return runResult{}, err
		}
		svc := policy.New(pol, st, time.Minute, log, policy.WithInstance(fmt.Sprintf("replica-%d", r)))
		srv := newInproc(func(s *grpc.Server) { pb.RegisterPolicyServiceServer(s, svc) })
		defer srv.stop()
		conn, err := srv.dial()
		if err != nil {
			return runResult{}, err
		}
		defer conn.Close()
		replicas[r] = pb.NewPolicyServiceClient(conn)

		// This replica hears from the aggregator through a client that
		// delays every snapshot by lag.
		followConn, err := aggSrv.dial()
		if err != nil {
			return runResult{}, err
		}
		defer followConn.Close()
		followers.Add(1)
		go func() {
			defer followers.Done()
			svc.Follow(followCtx, laggedClient{pb.NewAggregatorServiceClient(followConn), lag}, 10*time.Millisecond)
		}()
	}

	bestArm, bestP := 0, cfg.Probs[0]
	for a, p := range cfg.Probs {
		if p > bestP {
			bestArm, bestP = a, p
		}
	}

	var (
		next    atomic.Int64
		mu      sync.Mutex
		regret  float64
		bestN   int
		runErr  error
		workers sync.WaitGroup
	)
	start := time.Now()
	for w := range cfg.Workers {
		env, err := bandit.NewEnv(cfg.Probs, bandit.NewRNG(seed, bandit.StreamEnv+uint64(1000+w)))
		if err != nil {
			return runResult{}, err
		}
		workers.Add(1)
		go func() {
			defer workers.Done()
			var myRegret float64
			myBest := 0
			for next.Add(1) <= int64(cfg.Selections) {
				sel, err := replicas[w%cfg.Replicas].Select(ctx, &pb.SelectRequest{})
				if err != nil {
					mu.Lock()
					runErr = errors.Join(runErr, err)
					mu.Unlock()
					return
				}
				arm := int(sel.GetArm())
				myRegret += bestP - cfg.Probs[arm]
				if arm == bestArm {
					myBest++
				}
				if !sleep(ctx, cfg.RewardDelay) {
					return
				}
				if _, err := feedbackClient.Reward(ctx, &pb.RewardRequest{RequestId: sel.GetRequestId(), Reward: env.Pull(arm)}); err != nil {
					mu.Lock()
					runErr = errors.Join(runErr, err)
					mu.Unlock()
					return
				}
				if !sleep(ctx, cfg.Think) {
					return
				}
			}
			mu.Lock()
			regret += myRegret
			bestN += myBest
			mu.Unlock()
		}()
	}
	workers.Wait()
	if runErr != nil {
		return runResult{}, runErr
	}
	if err := ctx.Err(); err != nil {
		return runResult{}, err
	}
	elapsed := time.Since(start)
	return runResult{
		regret:    regret,
		bestShare: float64(bestN) / float64(cfg.Selections),
		rate:      float64(cfg.Selections) / elapsed.Seconds(),
	}, nil
}

// sleep waits d, or until ctx ends; it reports whether the full time passed.
func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	select {
	case <-time.After(d):
		return true
	case <-ctx.Done():
		return false
	}
}
