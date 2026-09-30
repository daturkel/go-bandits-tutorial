package ex2

import (
	"google.golang.org/grpc"
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
	// TODO
	return nil
}
