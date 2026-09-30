package rpcx_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	pb "banditlab/internal/gen/bandit/v1"
	"banditlab/internal/rpcx"
	"banditlab/internal/rpcx/rpcxtest"
	"banditlab/internal/store"
)

func TestFromStore(t *testing.T) {
	tests := []struct {
		err  error
		want codes.Code
	}{
		{nil, codes.OK},
		{store.ErrUnknownID, codes.NotFound},
		{fmt.Errorf("claim: %w", store.ErrAlreadyRewarded), codes.AlreadyExists},
		{context.DeadlineExceeded, codes.DeadlineExceeded},
		{context.Canceled, codes.Canceled},
		{errors.New("dial tcp 10.0.0.5:5432: connection refused"), codes.Unavailable},
	}
	for _, tc := range tests {
		got := rpcx.FromStore(tc.err)
		if status.Code(got) != tc.want {
			t.Errorf("FromStore(%v) = %v, want %v", tc.err, status.Code(got), tc.want)
		}
	}
	// The driver's message must not reach the client.
	if msg := status.Convert(rpcx.FromStore(errors.New("password=hunter2 refused"))).Message(); strings.Contains(msg, "hunter2") {
		t.Errorf("internal detail leaked: %q", msg)
	}
}

// stub is a policy service whose behaviour each test sets.
type stub struct {
	pb.UnimplementedPolicyServiceServer
	selectFn func(context.Context) (*pb.SelectResponse, error)
}

func (s *stub) Select(ctx context.Context, _ *pb.SelectRequest) (*pb.SelectResponse, error) {
	return s.selectFn(ctx)
}

func startStub(t *testing.T, log *slog.Logger, deadline time.Duration, fn func(context.Context) (*pb.SelectResponse, error)) pb.PolicyServiceClient {
	t.Helper()
	conn := rpcxtest.Start(t, func(s *grpc.Server) {
		pb.RegisterPolicyServiceServer(s, &stub{selectFn: fn})
	}, rpcx.ServerOptions(log, deadline)...)
	return pb.NewPolicyServiceClient(conn)
}

func TestPanicBecomesInternalAndServerSurvives(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, nil))
	calls := 0
	c := startStub(t, log, time.Second, func(context.Context) (*pb.SelectResponse, error) {
		calls++
		if calls == 1 {
			panic("boom")
		}
		return &pb.SelectResponse{Arm: 2}, nil
	})

	_, err := c.Select(context.Background(), &pb.SelectRequest{})
	if status.Code(err) != codes.Internal {
		t.Fatalf("first call: %v, want Internal", err)
	}
	if strings.Contains(err.Error(), "boom") {
		t.Errorf("panic value leaked to the client: %v", err)
	}
	if !strings.Contains(logs.String(), "boom") {
		t.Errorf("panic not logged: %s", logs.String())
	}
	resp, err := c.Select(context.Background(), &pb.SelectRequest{})
	if err != nil || resp.GetArm() != 2 {
		t.Errorf("second call = %v, %v; the server should still work", resp, err)
	}
}

func TestDefaultDeadlineIsAddedOnlyWhenMissing(t *testing.T) {
	var seen []time.Duration
	var mu sync.Mutex
	c := startStub(t, slog.New(slog.DiscardHandler), 3*time.Second, func(ctx context.Context) (*pb.SelectResponse, error) {
		d, ok := ctx.Deadline()
		mu.Lock()
		defer mu.Unlock()
		if !ok {
			seen = append(seen, -1)
		} else {
			seen = append(seen, time.Until(d))
		}
		return &pb.SelectResponse{}, nil
	})

	c.Select(context.Background(), &pb.SelectRequest{}) // no deadline from the client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c.Select(ctx, &pb.SelectRequest{}) // the client's own, longer deadline

	if len(seen) != 2 {
		t.Fatalf("saw %d calls", len(seen))
	}
	if seen[0] <= 0 || seen[0] > 3*time.Second {
		t.Errorf("call without a deadline got %v, want at most 3s", seen[0])
	}
	if seen[1] < 20*time.Second {
		t.Errorf("call with its own deadline saw %v, want it left alone", seen[1])
	}
}

func TestClientDeadlineReachesTheHandler(t *testing.T) {
	c := startStub(t, slog.New(slog.DiscardHandler), time.Minute, func(ctx context.Context) (*pb.SelectResponse, error) {
		<-ctx.Done() // a handler that waits until told to stop
		return nil, status.FromContextError(ctx.Err()).Err()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := c.Select(ctx, &pb.SelectRequest{})
	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("took %v to give up", took)
	}
}

func TestLoggingLevelsFollowTheStatusCode(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	results := []error{nil, status.Error(codes.NotFound, "x"), status.Error(codes.Unavailable, "x"), status.Error(codes.Unknown, "x")}
	i := 0
	c := startStub(t, log, time.Second, func(context.Context) (*pb.SelectResponse, error) {
		err := results[i]
		i++
		return &pb.SelectResponse{}, err
	})
	for range results {
		c.Select(context.Background(), &pb.SelectRequest{})
	}
	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	want := []string{"level=DEBUG", "level=DEBUG", "level=WARN", "level=ERROR"}
	for k, w := range want {
		if !strings.Contains(lines[k], w) {
			t.Errorf("line %d = %q, want %s", k, lines[k], w)
		}
	}
}

// Serve stops accepting work on cancel, lets a call in progress finish, and
// reports NOT_SERVING while it does.
func TestServeShutsDownGracefully(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- rpcx.Serve(ctx, ln, slog.New(slog.DiscardHandler), rpcx.Config{Grace: 5 * time.Second}, func(s *grpc.Server, _ *health.Server) {
			pb.RegisterPolicyServiceServer(s, &stub{selectFn: func(context.Context) (*pb.SelectResponse, error) {
				close(started)
				<-release
				return &pb.SelectResponse{Arm: 1}, nil
			}})
		})
	}()

	conn, err := grpc.NewClient(ln.Addr().String(), grpc.WithTransportCredentials(insecureCreds()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	result := make(chan *pb.SelectResponse, 1)
	go func() {
		r, _ := pb.NewPolicyServiceClient(conn).Select(context.Background(), &pb.SelectRequest{})
		result <- r
	}()
	<-started
	stop()

	select {
	case err := <-done:
		t.Fatalf("Serve returned (%v) while a call was still running", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	if err := <-done; err != nil {
		t.Errorf("Serve = %v, want nil after a clean shutdown", err)
	}
	if r := <-result; r.GetArm() != 1 {
		t.Errorf("the in-flight call got %v, want its answer", r)
	}
}

func TestServeGraceExpires(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	started := make(chan struct{})
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- rpcx.Serve(ctx, ln, slog.New(slog.DiscardHandler), rpcx.Config{Grace: 100 * time.Millisecond}, func(s *grpc.Server, _ *health.Server) {
			pb.RegisterPolicyServiceServer(s, &stub{selectFn: func(ctx context.Context) (*pb.SelectResponse, error) {
				close(started)
				<-ctx.Done() // only ends when the server forces the connection closed
				return nil, ctx.Err()
			}})
		})
	}()
	conn, _ := grpc.NewClient(ln.Addr().String(), grpc.WithTransportCredentials(insecureCreds()))
	defer conn.Close()
	go pb.NewPolicyServiceClient(conn).Select(context.Background(), &pb.SelectRequest{})
	<-started
	stop()
	if err := <-done; !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Serve = %v, want an error wrapping DeadlineExceeded", err)
	}
}

func TestServeReportsHealth(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- rpcx.Serve(ctx, ln, slog.New(slog.DiscardHandler), rpcx.Config{Grace: time.Second}, func(*grpc.Server, *health.Server) {})
	}()
	conn, _ := grpc.NewClient(ln.Addr().String(), grpc.WithTransportCredentials(insecureCreds()))
	defer conn.Close()
	resp, err := healthpb.NewHealthClient(conn).Check(context.Background(), &healthpb.HealthCheckRequest{})
	if err != nil || resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		t.Errorf("Check = %v, %v; want SERVING", resp, err)
	}
	stop()
	<-done
}
