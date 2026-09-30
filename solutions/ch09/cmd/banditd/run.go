package main

import (
	"context"
	"fmt"
	"io"
	"net"

	"banditlab/internal/bandit"
	"banditlab/internal/config"
	"banditlab/internal/logging"
	"banditlab/internal/server"
)

// run is the whole program apart from process-level concerns (signals, exit
// codes), so tests can start and stop it. Its inputs are explicit: arguments,
// an environment lookup, and where to write logs.
func run(ctx context.Context, args []string, getenv func(string) string, stderr io.Writer) error {
	cfg, err := config.Load(args, getenv, stderr)
	if err != nil {
		return err
	}
	log, err := logging.New(stderr, cfg.LogFormat, cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("%w: %w", config.ErrInvalid, err)
	}
	log = log.With("service", "banditd") // added to every record from here on

	pol, err := bandit.NewSnapshotPolicy(cfg.Policy, cfg.Arms, bandit.NewRNG(cfg.Seed, bandit.StreamPolicy))
	if err != nil {
		return fmt.Errorf("%w: %w", config.ErrInvalid, err)
	}
	srv := server.New(pol, log, server.WithSelectDelay(cfg.SelectDelay))

	// Listen before logging, so a port that is already taken fails here with
	// a clear error instead of after we claim to be serving.
	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return err
	}
	log.Info("listening",
		"addr", ln.Addr().String(),
		"policy", pol.Name(),
		"arms", cfg.Arms,
		"select_delay", cfg.SelectDelay,
	)
	return server.Serve(ctx, ln, srv.Handler(), log, server.ServeOptions{
		ShutdownGrace: cfg.ShutdownGrace,
		OnShutdown:    srv.SetDraining,
		DrainDelay:    cfg.DrainDelay,
	})
}
