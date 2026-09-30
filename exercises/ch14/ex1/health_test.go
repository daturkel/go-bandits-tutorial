package ex1

import (
	"context"
	"net"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

func client(t *testing.T, hs *health.Server) healthpb.HealthClient {
	t.Helper()
	lis := bufconn.Listen(1 << 16)
	srv := grpc.NewServer()
	healthpb.RegisterHealthServer(srv, hs)
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient("passthrough:///x",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return healthpb.NewHealthClient(conn)
}

func TestCheck(t *testing.T) {
	hs := health.NewServer()
	hs.SetServingStatus("up", healthpb.HealthCheckResponse_SERVING)
	hs.SetServingStatus("down", healthpb.HealthCheckResponse_NOT_SERVING)
	c := client(t, hs)
	ctx := context.Background()

	if err := Check(ctx, c, "up"); err != nil {
		t.Errorf("serving service: %v, want nil", err)
	}
	err := Check(ctx, c, "down")
	if err == nil || !strings.Contains(err.Error(), "NOT_SERVING") {
		t.Errorf("not-serving service: %v, want an error mentioning NOT_SERVING", err)
	}
	if err := Check(ctx, c, "nobody"); err == nil {
		t.Error("an unknown service must not count as healthy")
	}
}

func TestCheckWholeProcess(t *testing.T) {
	c := client(t, health.NewServer()) // the empty name is SERVING by default
	if err := Check(context.Background(), c, ""); err != nil {
		t.Errorf("whole process: %v, want nil", err)
	}
}

func TestCheckHonoursTheContext(t *testing.T) {
	c := client(t, health.NewServer())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Check(ctx, c, ""); err == nil {
		t.Error("a cancelled context must fail the check")
	}
}
