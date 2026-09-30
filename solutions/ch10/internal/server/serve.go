package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// ServeOptions tunes Serve.
type ServeOptions struct {
	// ShutdownGrace is how long in-flight requests get to finish once ctx is
	// cancelled. After that, remaining connections are closed. Zero means
	// close immediately.
	ShutdownGrace time.Duration

	// OnShutdown, if set, is called as soon as ctx is cancelled. Use it to
	// start failing health checks.
	OnShutdown func()

	// DrainDelay is how long to keep accepting connections after OnShutdown,
	// so a load balancer polling the health check has time to notice and stop
	// routing here. Zero means stop accepting at once.
	DrainDelay time.Duration
}

// Serve runs h on ln until ctx is cancelled, then shuts down gracefully:
// it calls opts.OnShutdown, waits opts.DrainDelay, stops accepting
// connections, lets requests already in progress finish (up to
// opts.ShutdownGrace), and returns.
//
// It returns nil after a clean shutdown, an error wrapping
// context.DeadlineExceeded if the grace period ran out with requests still
// running, or the serving error if the listener failed.
func Serve(ctx context.Context, ln net.Listener, h http.Handler, log *slog.Logger, opts ServeOptions) error {
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
		return err // the listener failed before anyone asked us to stop
	case <-ctx.Done():
	}

	log.Info("shutting down", "drain_delay", opts.DrainDelay, "grace", opts.ShutdownGrace)
	if opts.OnShutdown != nil {
		opts.OnShutdown()
	}
	time.Sleep(opts.DrainDelay)
	// ctx is already cancelled, so the deadline must hang off a context that
	// keeps its values but not its cancellation.
	shutCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), opts.ShutdownGrace)
	defer cancel()

	err := srv.Shutdown(shutCtx)
	if err != nil {
		// Grace ran out. Shutdown left the stragglers running; cut them off.
		log.Warn("grace period expired, closing remaining connections")
		srv.Close()
		err = fmt.Errorf("shutdown: %w", err)
	} else {
		log.Info("shutdown complete")
	}
	if serveErr := <-errc; !errors.Is(serveErr, http.ErrServerClosed) && err == nil {
		err = serveErr
	}
	return err
}
