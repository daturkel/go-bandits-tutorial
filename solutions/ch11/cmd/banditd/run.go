package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"runtime"
	"sync"
	"time"

	"banditlab/internal/bandit"
	"banditlab/internal/config"
	"banditlab/internal/logging"
	"banditlab/internal/metrics"
	"banditlab/internal/server"
	"banditlab/internal/store"
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

	st, err := openStore(ctx, cfg, log)
	if err != nil {
		return err
	}
	defer st.Close()

	srv := server.New(pol, st, log,
		server.WithSelectDelay(cfg.SelectDelay),
		server.WithPendingTTL(cfg.PendingTTL),
		server.WithMetrics(metrics.New()),
	)
	if err := srv.Restore(ctx); err != nil {
		return err
	}

	// Sweep expired selections in the background, for as long as we serve.
	var wg sync.WaitGroup
	sweepCtx, stopSweeping := context.WithCancel(ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()
		srv.ExpireLoop(sweepCtx, cfg.ExpireInterval, cfg.ExpireReward)
	}()
	defer func() {
		stopSweeping()
		wg.Wait() // do not close the store under a sweep in progress
	}()

	// Listen before logging, so a port that is already taken fails here with
	// a clear error instead of after we claim to be serving.
	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return err
	}
	if cfg.DebugAddr != "" {
		if err := startDebugServer(sweepCtx, &wg, cfg, log); err != nil {
			ln.Close()
			return err
		}
	}
	log.Info("listening",
		"addr", ln.Addr().String(),
		"policy", pol.Name(),
		"arms", cfg.Arms,
		"pending_ttl", cfg.PendingTTL,
		"select_delay", cfg.SelectDelay,
	)
	return server.Serve(ctx, ln, srv.Handler(), log, server.ServeOptions{
		ShutdownGrace: cfg.ShutdownGrace,
		OnShutdown:    srv.SetDraining,
		DrainDelay:    cfg.DrainDelay,
	})
}

// openStore connects to PostgreSQL if a URL is configured and otherwise keeps
// everything in memory.
func openStore(ctx context.Context, cfg config.Config, log *slog.Logger) (store.Store, error) {
	if cfg.DatabaseURL == "" {
		log.Warn("no database configured: state is kept in memory and lost on exit")
		return store.NewMemory(cfg.Arms), nil
	}
	st, err := store.OpenPostgres(ctx, cfg.DatabaseURL, cfg.Arms)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	log.Info("database connected")
	return st, nil
}

// startDebugServer serves pprof on its own listener. It shares ctx, so it
// stops when the daemon does, and registers with wg so run can wait for it.
func startDebugServer(ctx context.Context, wg *sync.WaitGroup, cfg config.Config, log *slog.Logger) error {
	ln, err := net.Listen("tcp", cfg.DebugAddr)
	if err != nil {
		return fmt.Errorf("debug listener: %w", err)
	}
	// Mutex and block profiles are off by default because they cost a little
	// on every contended lock; the debug server is when you want them.
	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(1000)

	log = log.With("server", "debug")
	log.Info("listening", "addr", ln.Addr().String(), "endpoints", "/debug/pprof/")
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := server.Serve(ctx, ln, server.DebugHandler(), log, server.ServeOptions{
			ShutdownGrace: cfg.ShutdownGrace,
			WriteTimeout:  2 * time.Minute, // a 30-second CPU profile must fit
		})
		if err != nil {
			log.Error("debug server stopped", "err", err)
		}
	}()
	return nil
}
