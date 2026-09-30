package rpcx

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// An interceptor is gRPC's middleware: a function that wraps every call to a
// service. Unary interceptors wrap ordinary request/response calls; stream
// interceptors wrap the whole life of a stream.

// UnaryLogging logs one line per unary call: method, status code, duration.
// Codes a client can cause (NotFound, InvalidArgument, ...) log at debug; the
// ones that mean the service has a problem log at warn or error.
func UnaryLogging(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := next(ctx, req)
		logCall(ctx, log, info.FullMethod, err, time.Since(start))
		return resp, err
	}
}

// StreamLogging is UnaryLogging for streaming calls; the duration is the
// lifetime of the stream.
func StreamLogging(log *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, next grpc.StreamHandler) error {
		start := time.Now()
		err := next(srv, ss)
		logCall(ss.Context(), log, info.FullMethod, err, time.Since(start))
		return err
	}
}

func logCall(ctx context.Context, log *slog.Logger, method string, err error, took time.Duration) {
	code := status.Code(err)
	level := slog.LevelDebug
	switch code {
	case codes.OK, codes.NotFound, codes.AlreadyExists, codes.InvalidArgument, codes.Canceled:
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		level = slog.LevelWarn
	default:
		level = slog.LevelError
	}
	log.Log(ctx, level, "rpc", "method", method, "code", code.String(), "duration", took.Round(time.Microsecond))
}

// UnaryRecover turns a panic in a handler into an Internal error, so one bad
// request cannot take the process down. The stack goes to the log, not to the
// client.
func UnaryRecover(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic in rpc handler", "method", info.FullMethod, "panic", r, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return next(ctx, req)
	}
}

// StreamRecover is UnaryRecover for streaming handlers.
func StreamRecover(log *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, next grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic in rpc handler", "method", info.FullMethod, "panic", r, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return next(srv, ss)
	}
}

// UnaryDefaultDeadline gives calls that arrive without a deadline one of d. A
// client that forgets to set a deadline would otherwise be able to hold a
// handler (and whatever it holds, such as a database connection) forever.
func UnaryDefaultDeadline(d time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, d)
			defer cancel()
		}
		return next(ctx, req)
	}
}

// ServerOptions returns the standard interceptor chain, outermost first:
// recovery, then logging (so a recovered panic is logged as the Internal
// error it becomes), then the default deadline for unary calls.
func ServerOptions(log *slog.Logger, defaultDeadline time.Duration) []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(UnaryRecover(log), UnaryLogging(log), UnaryDefaultDeadline(defaultDeadline)),
		grpc.ChainStreamInterceptor(StreamRecover(log), StreamLogging(log)),
	}
}
