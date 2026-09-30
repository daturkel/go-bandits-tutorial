package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"banditlab/bandit"
	"banditlab/server"
)

// serveCmd runs the HTTP service until ctx is cancelled.
func serveCmd(ctx context.Context, args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("banditsim serve", flag.ContinueOnError)
	addr := fs.String("addr", "localhost:8080", "address to listen on")
	spec := fs.String("policy", "thompson", "policy spec, e.g. thompson, ucb1, epsgreedy:0.05")
	arms := fs.Int("arms", 3, "number of arms")
	seed := fs.Uint64("seed", 1, "random seed for the policy")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	pol, err := bandit.NewSnapshotPolicy(*spec, *arms, bandit.NewRNG(*seed, bandit.StreamPolicy))
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}
	log := slog.New(slog.NewTextHandler(stderr, nil))

	// Listen before logging, so a port that is already taken fails here with
	// a clear error instead of after we claim to be serving.
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}
	log.Info("listening", "addr", ln.Addr().String(), "policy", pol.Name(), "arms", *arms)
	return serve(ctx, ln, server.New(pol, log).Handler(), log)
}

// serve runs h on ln until ctx is cancelled.
func serve(ctx context.Context, ln net.Listener, h http.Handler, log *slog.Logger) error {
	srv := &http.Server{
		Handler: h,
		// http.ListenAndServe has no timeouts at all: a client that connects
		// and sends nothing would hold a connection forever.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		log.Info("stopping")
		// Abrupt: requests in flight are cut off. Chapter 9 drains them.
		if err := srv.Close(); err != nil {
			return err
		}
		if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
