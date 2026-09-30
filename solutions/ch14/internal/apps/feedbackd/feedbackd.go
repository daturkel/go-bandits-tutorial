// Package feedbackd is the feedback daemon: it receives rewards, records them
// in the store and reports the counts to the aggregator.
package feedbackd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"

	"banditlab/internal/daemon"
	pb "banditlab/internal/gen/bandit/v1"
	"banditlab/internal/rpcx"
	"banditlab/internal/services/feedback"
)

// Run starts the daemon and returns when ctx is cancelled and shutdown is done.
func Run(ctx context.Context, args []string, getenv func(string) string, stderr io.Writer) error {
	env := daemon.NewEnv(getenv)
	var common daemon.Common
	fs := flag.NewFlagSet("feedbackd", flag.ContinueOnError)
	fs.SetOutput(stderr)
	finish := common.Register(fs, env, "localhost:9091")
	arms := fs.Int("arms", env.Int("BANDIT_ARMS", 3), "number of arms (env BANDIT_ARMS)")
	dbURL := fs.String("database-url", env.Str("BANDIT_DATABASE_URL", ""), "PostgreSQL `url`; empty keeps state in memory only (env BANDIT_DATABASE_URL)")
	aggAddr := fs.String("aggregator", env.Str("BANDIT_AGGREGATOR", ""), "aggregator `address` to report counts to; empty reports nowhere (env BANDIT_AGGREGATOR)")
	source := fs.String("source", env.Str("BANDIT_SOURCE", "feedbackd"), "name this instance reports as (env BANDIT_SOURCE)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := finish(); err != nil {
		return err
	}
	if err := env.Err(); err != nil {
		return err
	}
	if *arms < 1 {
		return fmt.Errorf("%w: arms must be at least 1", daemon.ErrInvalid)
	}

	log, err := common.Logger(stderr, "feedbackd")
	if err != nil {
		return err
	}
	st, err := daemon.OpenStore(ctx, *dbURL, *arms, log)
	if err != nil {
		return err
	}
	defer st.Close()

	var reporter feedback.Reporter
	if *aggAddr != "" {
		// NewClient does not connect yet; the first call does, and gRPC
		// reconnects on its own if the aggregator restarts.
		conn, err := grpc.NewClient(*aggAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("%w: aggregator address: %w", daemon.ErrInvalid, err)
		}
		defer conn.Close()
		reporter = feedback.NewReporter(pb.NewAggregatorServiceClient(conn), *source)
	} else {
		log.Warn("no aggregator configured: rewards are stored but nobody is told")
	}
	svc := feedback.New(st, *arms, reporter, log)

	ln, err := net.Listen("tcp", common.Addr)
	if err != nil {
		return err
	}
	log.Info("listening", "addr", ln.Addr().String(), "arms", *arms, "aggregator", *aggAddr)
	cfg := rpcx.Config{Grace: common.Grace, ServerOptions: rpcx.ServerOptions(log, common.Deadline)}
	return rpcx.Serve(ctx, ln, log, cfg,
		func(s *grpc.Server, _ *health.Server) { pb.RegisterFeedbackServiceServer(s, svc) })
}
