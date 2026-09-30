package ex2

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MaxInFlight returns a unary server interceptor that lets at most n calls
// run at once. A call that arrives when n are already running is rejected
// immediately with codes.ResourceExhausted, without calling the handler:
// telling a client "busy, try later" at once is kinder to everyone than
// queueing until the client's deadline passes.
//
// It must be safe for concurrent use, and a call that panics or returns an
// error must still give its slot back.
func MaxInFlight(n int) grpc.UnaryServerInterceptor {
	slots := make(chan struct{}, n) // a buffered channel used as a counting semaphore
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }() // runs on return and on panic
			return next(ctx, req)
		default:
			return nil, status.Error(codes.ResourceExhausted, "too many calls in progress, try again")
		}
	}
}
