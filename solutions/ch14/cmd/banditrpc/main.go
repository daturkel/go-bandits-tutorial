// Command banditrpc calls the gRPC services from the command line.
//
//	banditrpc [-addr target] [-lb round_robin] select [N]
//	banditrpc [-addr host:port] stats
//	banditrpc [-addr host:port] reward REQUEST_ID REWARD
//	banditrpc [-addr host:port] batch          (lines "REQUEST_ID REWARD" on stdin)
//	banditrpc [-addr host:port] watch [N]      (print N snapshots, default 1)
//	banditrpc [-addr host:port] health
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"
	"google.golang.org/grpc/status"

	pb "banditlab/internal/gen/bandit/v1"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	stop()
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, "banditrpc:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("banditrpc", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", "localhost:9090", "service `target`: host:port, dns:///name:port (every address the name resolves to), or a comma-separated list of host:port")
	lb := fs.String("lb", "", "load-balancing policy when the target names several backends: round_robin (default: pick_first)")
	timeout := fs.Duration("timeout", 5*time.Second, "deadline for each call")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return errors.New("usage: banditrpc [-addr host:port] select|stats|reward|batch|watch|health")
	}
	target, dialOpts := dialTarget(*addr)
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if *lb != "" {
		// A service config chooses how calls are spread over the backends a
		// target resolves to. The default, pick_first, uses one and sticks
		// with it, which for a long-lived HTTP/2 connection means one replica
		// gets everything.
		cfg := fmt.Sprintf(`{"loadBalancingConfig":[{%q:{}}]}`, *lb)
		dialOpts = append(dialOpts, grpc.WithDefaultServiceConfig(cfg))
	}
	conn, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Every call gets a deadline. Without one, a call to a service that has
	// stopped answering would wait forever.
	callCtx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	switch cmd, cargs := rest[0], rest[1:]; cmd {
	case "select":
		n := 1
		if len(cargs) == 1 {
			var err error
			if n, err = strconv.Atoi(cargs[0]); err != nil || n < 1 {
				return errors.New("usage: banditrpc select [N]")
			}
		}
		for range n {
			resp, err := pb.NewPolicyServiceClient(conn).Select(callCtx, &pb.SelectRequest{})
			if err != nil {
				return explain(err)
			}
			fmt.Fprintf(stdout, "request_id=%s arm=%d instance=%s\n", resp.GetRequestId(), resp.GetArm(), resp.GetInstance())
		}
	case "stats":
		resp, err := pb.NewPolicyServiceClient(conn).Stats(callCtx, &pb.StatsRequest{})
		if err != nil {
			return explain(err)
		}
		fmt.Fprintf(stdout, "policy=%s instance=%s snapshot=%d\n", resp.GetPolicy(), resp.GetInstance(), resp.GetSnapshotVersion())
		for _, a := range resp.GetArms() {
			fmt.Fprintf(stdout, "  arm %d: pulls=%d mean=%.3f pending=%d\n", a.GetArm(), a.GetPulls(), a.GetMean(), a.GetPending())
		}
	case "reward":
		if len(cargs) != 2 {
			return errors.New("usage: banditrpc reward REQUEST_ID REWARD")
		}
		r, err := strconv.ParseFloat(cargs[1], 64)
		if err != nil {
			return err
		}
		resp, err := pb.NewFeedbackServiceClient(conn).Reward(callCtx, &pb.RewardRequest{RequestId: cargs[0], Reward: r})
		if err != nil {
			return explain(err)
		}
		fmt.Fprintf(stdout, "recorded on arm %d\n", resp.GetArm())
	case "batch":
		stream, err := pb.NewFeedbackServiceClient(conn).RewardBatch(callCtx)
		if err != nil {
			return explain(err)
		}
		sc := bufio.NewScanner(stdin)
		for sc.Scan() {
			id, val, ok := strings.Cut(strings.TrimSpace(sc.Text()), " ")
			r, perr := strconv.ParseFloat(val, 64)
			if !ok || perr != nil {
				return fmt.Errorf("bad line %q: want REQUEST_ID REWARD", sc.Text())
			}
			if err := stream.Send(&pb.RewardBatchRequest{RequestId: id, Reward: r}); err != nil {
				return explain(err)
			}
		}
		sum, err := stream.CloseAndRecv()
		if err != nil {
			return explain(err)
		}
		fmt.Fprintf(stdout, "accepted=%d rejected=%d\n", sum.GetAccepted(), sum.GetRejected())
	case "watch":
		n := 1
		if len(cargs) == 1 {
			if n, err = strconv.Atoi(cargs[0]); err != nil {
				return err
			}
		}
		// A watch is long-lived, so it uses the caller's context, not the
		// per-call timeout above.
		stream, err := pb.NewAggregatorServiceClient(conn).Watch(ctx, &pb.WatchRequest{})
		if err != nil {
			return explain(err)
		}
		for range n {
			snap, err := stream.Recv()
			if err != nil {
				return explain(err)
			}
			fmt.Fprintf(stdout, "version=%d", snap.GetVersion())
			for _, a := range snap.GetTotals() {
				fmt.Fprintf(stdout, " arm%d=%d/%.0f", a.GetArm(), a.GetPulls(), a.GetRewardSum())
			}
			fmt.Fprintln(stdout)
		}
	case "health":
		resp, err := healthpb.NewHealthClient(conn).Check(callCtx, &healthpb.HealthCheckRequest{})
		if err != nil {
			return explain(err)
		}
		fmt.Fprintln(stdout, resp.GetStatus())
		// The exit status is what Docker, Compose and Kubernetes health
		// checks look at, so anything but SERVING must fail.
		if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
			return fmt.Errorf("%s is %s", *addr, resp.GetStatus())
		}
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
	return nil
}

// dialTarget turns the -addr flag into a gRPC target. A comma-separated list
// has no built-in scheme in gRPC-Go, so it gets a resolver that always
// returns exactly those addresses; anything else is passed through, and gRPC
// resolves it (dns:///name:port yields every address the name has).
func dialTarget(addr string) (string, []grpc.DialOption) {
	if !strings.Contains(addr, ",") {
		return addr, nil
	}
	var addrs []resolver.Address
	for a := range strings.SplitSeq(addr, ",") {
		addrs = append(addrs, resolver.Address{Addr: strings.TrimSpace(a)})
	}
	r := manual.NewBuilderWithScheme("static")
	r.InitialState(resolver.State{Addresses: addrs})
	return "static:///backends", []grpc.DialOption{grpc.WithResolvers(r)}
}

// explain shows a gRPC error as "Code: message", which is what you want to
// read at a terminal.
func explain(err error) error {
	if s, ok := status.FromError(err); ok {
		return fmt.Errorf("%s: %s", s.Code(), s.Message())
	}
	return err
}
