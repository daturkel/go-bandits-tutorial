// Package rpcxtest starts gRPC servers inside a test process. Calls travel
// through an in-memory pipe (bufconn) instead of a TCP port, so the full
// stack runs, including serialisation, status codes, deadlines and
// interceptors, with no port to pick and nothing to clean up.
package rpcxtest

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// Start serves whatever register installs and returns a client connection to
// it. Both are closed when the test ends.
func Start(t testing.TB, register func(*grpc.Server), opts ...grpc.ServerOption) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer(opts...)
	register(srv)
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}
