package ex1

import (
	"context"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Check asks a gRPC health service whether the named service is serving
// (use the empty name for the process as a whole). It returns nil only if the
// answer is SERVING. Any other status, and any error from the call, must come
// back as an error, because a container health check treats "no error" as
// healthy and a check that says yes to NOT_SERVING keeps traffic flowing to a
// service that is shutting down.
//
// The error for a non-SERVING answer should mention the status.
func Check(ctx context.Context, c healthpb.HealthClient, service string) error {
	// TODO
	return nil
}
