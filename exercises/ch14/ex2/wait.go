package ex2

import (
	"context"
	"time"
)

// WaitHealthy calls check every interval until it returns nil, and then
// returns nil. It calls check once immediately, without waiting. If ctx ends
// first it returns an error that wraps ctx.Err() and the last error check
// returned (so the caller can see why the service never became healthy).
//
// This is what `depends_on: condition: service_healthy` does for containers
// and what the `until ... health` loops in the demo scripts do.
func WaitHealthy(ctx context.Context, interval time.Duration, check func(context.Context) error) error {
	// TODO
	return nil
}
