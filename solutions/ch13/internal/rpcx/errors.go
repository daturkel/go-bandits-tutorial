// Package rpcx holds what every gRPC service in the system needs: mapping
// errors to status codes, interceptors, and a server runner with graceful
// shutdown.
package rpcx

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"banditlab/internal/store"
)

// FromStore turns an error from the store into the status a client should
// see. Callers can then test the code (codes.NotFound and so on) instead of
// matching strings, and internal details such as driver messages stay in the
// server's log.
//
//	unknown or expired request id     NotFound
//	reward already recorded           AlreadyExists
//	the caller's deadline passed      DeadlineExceeded
//	the caller gave up                Canceled
//	anything else                     Unavailable (retrying may help)
func FromStore(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, store.ErrUnknownID):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, store.ErrAlreadyRewarded):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request cancelled")
	default:
		return status.Error(codes.Unavailable, "storage unavailable, try again")
	}
}
