package rpcx

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Config tunes Serve.
type Config struct {
	// Grace is how long calls in progress get to finish once shutdown starts.
	Grace time.Duration

	// ServerOptions are passed to grpc.NewServer (interceptors, limits).
	ServerOptions []grpc.ServerOption

	// OnStopping, if set, is called when shutdown starts, before the server
	// waits for calls to finish. Services with long-lived streams use it to
	// end them: GracefulStop waits for streams like any other call, and a
	// stream that never ends would use up the whole grace period.
	OnStopping func()
}

// Serve runs a gRPC server on ln until ctx is cancelled and then shuts it
// down gracefully: the health service starts reporting NOT_SERVING, then
// GracefulStop lets calls in progress finish. If they have not after grace,
// Stop closes the connections and Serve returns an error wrapping
// context.DeadlineExceeded.
//
// register receives the server and the health service, and should register
// the caller's services on the server. Every service name it wants reported
// healthy should be set SERVING on the health server; Serve also sets the
// empty name (the whole process).
func Serve(ctx context.Context, ln net.Listener, log *slog.Logger, cfg Config, register func(*grpc.Server, *health.Server)) error {
	srv := grpc.NewServer(cfg.ServerOptions...)
	hs := health.NewServer()
	healthpb.RegisterHealthServer(srv, hs)
	register(srv, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		return err // the listener failed before we were asked to stop
	case <-ctx.Done():
	}

	log.Info("stopping", "grace", cfg.Grace)
	hs.Shutdown() // every service now reports NOT_SERVING
	if cfg.OnStopping != nil {
		cfg.OnStopping()
	}
	stopped := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
		return nil
	case <-time.After(cfg.Grace):
		srv.Stop()
		<-stopped
		return fmt.Errorf("rpc calls did not finish within %v: %w", cfg.Grace, context.DeadlineExceeded)
	}
}
