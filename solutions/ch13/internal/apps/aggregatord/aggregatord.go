// Package aggregatord is the aggregator daemon: it merges counts from the
// feedback services and streams the result to the policy services.
package aggregatord

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"

	"banditlab/internal/daemon"
	pb "banditlab/internal/gen/bandit/v1"
	"banditlab/internal/rpcx"
	"banditlab/internal/services/aggregator"
)

// Run starts the daemon and returns when ctx is cancelled and shutdown is done.
func Run(ctx context.Context, args []string, getenv func(string) string, stderr io.Writer) error {
	env := daemon.NewEnv(getenv)
	var common daemon.Common
	fs := flag.NewFlagSet("aggregatord", flag.ContinueOnError)
	fs.SetOutput(stderr)
	finish := common.Register(fs, env, "localhost:9092")
	arms := fs.Int("arms", env.Int("BANDIT_ARMS", 3), "number of arms (env BANDIT_ARMS)")
	dbURL := fs.String("database-url", env.Str("BANDIT_DATABASE_URL", ""), "PostgreSQL `url`; the truth this daemon reconciles against (env BANDIT_DATABASE_URL)")
	reconcile := fs.Duration("reconcile-interval", env.Duration("BANDIT_RECONCILE_INTERVAL", 10*time.Second), "how often to reload the totals from the database (env BANDIT_RECONCILE_INTERVAL)")
	expireEvery := fs.Duration("expire-interval", env.Duration("BANDIT_EXPIRE_INTERVAL", 30*time.Second), "how often expired selections are swept (env BANDIT_EXPIRE_INTERVAL)")
	expireReward := fs.String("expire-reward", env.Str("BANDIT_EXPIRE_REWARD", ""), "count unrewarded expired selections as this `reward`; empty ignores them (env BANDIT_EXPIRE_REWARD)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := finish(); err != nil {
		return err
	}
	if err := env.Err(); err != nil {
		return err
	}
	var implicit *float64
	if *expireReward != "" {
		r, err := strconv.ParseFloat(*expireReward, 64)
		if err != nil || !(r >= 0 && r <= 1) {
			return fmt.Errorf("%w: expire reward %q must be a number between 0 and 1", daemon.ErrInvalid, *expireReward)
		}
		implicit = &r
	}
	if *arms < 1 || *reconcile <= 0 || *expireEvery <= 0 {
		return fmt.Errorf("%w: arms must be at least 1 and intervals positive", daemon.ErrInvalid)
	}

	log, err := common.Logger(stderr, "aggregatord")
	if err != nil {
		return err
	}
	st, err := daemon.OpenStore(ctx, *dbURL, *arms, log)
	if err != nil {
		return err
	}
	defer st.Close()

	svc := aggregator.New(*arms, log)
	var wg sync.WaitGroup
	loopCtx, stopLoops := context.WithCancel(ctx)
	wg.Add(2)
	go func() { defer wg.Done(); svc.Reconcile(loopCtx, st, *reconcile) }()
	go func() { defer wg.Done(); svc.ExpireLoop(loopCtx, st, *expireEvery, implicit) }()
	defer func() {
		stopLoops()
		wg.Wait()
	}()

	ln, err := net.Listen("tcp", common.Addr)
	if err != nil {
		return err
	}
	log.Info("listening", "addr", ln.Addr().String(), "arms", *arms)
	cfg := rpcx.Config{Grace: common.Grace, ServerOptions: rpcx.ServerOptions(log, common.Deadline), OnStopping: svc.Close}
	return rpcx.Serve(ctx, ln, log, cfg,
		func(s *grpc.Server, _ *health.Server) { pb.RegisterAggregatorServiceServer(s, svc) })
}
