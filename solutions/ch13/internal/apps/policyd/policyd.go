// Package policyd is the policy daemon: it answers select calls and follows
// the aggregator's snapshots.
package policyd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"

	"banditlab/internal/bandit"
	"banditlab/internal/daemon"
	pb "banditlab/internal/gen/bandit/v1"
	"banditlab/internal/rpcx"
	"banditlab/internal/services/policy"
)

// Run starts the daemon and returns when ctx is cancelled and shutdown is done.
func Run(ctx context.Context, args []string, getenv func(string) string, stderr io.Writer) error {
	env := daemon.NewEnv(getenv)
	var common daemon.Common
	fs := flag.NewFlagSet("policyd", flag.ContinueOnError)
	fs.SetOutput(stderr)
	finish := common.Register(fs, env, "localhost:9090")
	spec := fs.String("policy", env.Str("BANDIT_POLICY", "thompson"), "policy `spec`: thompson, ucb1, epsgreedy[:eps] (env BANDIT_POLICY)")
	arms := fs.Int("arms", env.Int("BANDIT_ARMS", 3), "number of arms (env BANDIT_ARMS)")
	seed := fs.Uint64("seed", env.Uint64("BANDIT_SEED", 1), "policy random seed (env BANDIT_SEED)")
	dbURL := fs.String("database-url", env.Str("BANDIT_DATABASE_URL", ""), "PostgreSQL `url`; empty keeps state in memory only (env BANDIT_DATABASE_URL)")
	aggAddr := fs.String("aggregator", env.Str("BANDIT_AGGREGATOR", ""), "aggregator `address` to follow; empty uses only what the database held at start (env BANDIT_AGGREGATOR)")
	ttl := fs.Duration("pending-ttl", env.Duration("BANDIT_PENDING_TTL", 10*time.Minute), "how long a selection waits for its reward (env BANDIT_PENDING_TTL)")
	retry := fs.Duration("retry", env.Duration("BANDIT_RETRY", time.Second), "wait before resubscribing to a lost aggregator (env BANDIT_RETRY)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := finish(); err != nil {
		return err
	}
	if err := env.Err(); err != nil {
		return err
	}
	if *arms < 1 || *ttl <= 0 || *retry <= 0 {
		return fmt.Errorf("%w: arms must be at least 1 and durations positive", daemon.ErrInvalid)
	}
	pol, err := bandit.NewSnapshotPolicy(*spec, *arms, bandit.NewRNG(*seed, bandit.StreamPolicy))
	if err != nil {
		return fmt.Errorf("%w: %w", daemon.ErrInvalid, err)
	}

	log, err := common.Logger(stderr, "policyd")
	if err != nil {
		return err
	}
	st, err := daemon.OpenStore(ctx, *dbURL, *arms, log)
	if err != nil {
		return err
	}
	defer st.Close()

	svc := policy.New(pol, st, *ttl, log)
	// Start from what the store holds, so a restarted instance is correct
	// before the aggregator's first snapshot arrives.
	totals, err := st.Totals(ctx)
	if err != nil {
		return fmt.Errorf("load totals: %w", err)
	}
	if err := svc.Apply(ctx, 0, totals); err != nil {
		return err
	}

	var wg sync.WaitGroup
	followCtx, stopFollowing := context.WithCancel(ctx)
	defer func() {
		stopFollowing()
		wg.Wait()
	}()
	if *aggAddr != "" {
		conn, err := grpc.NewClient(*aggAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("%w: aggregator address: %w", daemon.ErrInvalid, err)
		}
		defer conn.Close()
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc.Follow(followCtx, pb.NewAggregatorServiceClient(conn), *retry)
		}()
	} else {
		log.Warn("no aggregator configured: this instance will not learn from rewards after it starts")
	}

	ln, err := net.Listen("tcp", common.Addr)
	if err != nil {
		return err
	}
	log.Info("listening", "addr", ln.Addr().String(), "policy", pol.Name(), "arms", *arms, "aggregator", *aggAddr)
	cfg := rpcx.Config{Grace: common.Grace, ServerOptions: rpcx.ServerOptions(log, common.Deadline)}
	return rpcx.Serve(ctx, ln, log, cfg,
		func(s *grpc.Server, _ *health.Server) { pb.RegisterPolicyServiceServer(s, svc) })
}
