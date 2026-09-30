package apps_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"banditlab/internal/apps/aggregatord"
	"banditlab/internal/apps/feedbackd"
	"banditlab/internal/apps/policyd"
	"banditlab/internal/daemon/daemontest"
	pb "banditlab/internal/gen/bandit/v1"
	"banditlab/internal/store/storetest"
)

func dial(t *testing.T, addr string) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// The whole system as separate daemons on real TCP ports sharing one
// PostgreSQL database: an aggregator, a feedback service and two policy
// services. A reward sent to the feedback service must reach the policy
// service that never saw the selection.
func TestServicesOverTCP(t *testing.T) {
	dbURL := storetest.DatabaseURL(t)
	u, err := url.Parse(dbURL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", storetest.NewSchema(t, dbURL))
	u.RawQuery = q.Encode()
	db := "-database-url=" + u.String()

	agg := daemontest.Start(t, aggregatord.Run, daemontest.NoEnv, "-addr", "127.0.0.1:0", db, "-reconcile-interval", "50ms")
	fb := daemontest.Start(t, feedbackd.Run, daemontest.NoEnv, "-addr", "127.0.0.1:0", db, "-aggregator", agg.Addr)
	polA := daemontest.Start(t, policyd.Run, daemontest.NoEnv, "-addr", "127.0.0.1:0", db, "-aggregator", agg.Addr, "-policy", "ucb1", "-retry", "20ms")
	polB := daemontest.Start(t, policyd.Run, daemontest.NoEnv, "-addr", "127.0.0.1:0", db, "-aggregator", agg.Addr, "-policy", "ucb1", "-retry", "20ms")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	a := pb.NewPolicyServiceClient(dial(t, polA.Addr))
	b := pb.NewPolicyServiceClient(dial(t, polB.Addr))
	f := pb.NewFeedbackServiceClient(dial(t, fb.Addr))

	for range 12 {
		sel, err := a.Select(ctx, &pb.SelectRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Reward(ctx, &pb.RewardRequest{RequestId: sel.GetRequestId(), Reward: 1}); err != nil {
			t.Fatal(err)
		}
	}
	for {
		stats, err := b.Stats(ctx, &pb.StatsRequest{})
		if err != nil {
			t.Fatal(err)
		}
		var pulls int64
		for _, arm := range stats.GetArms() {
			pulls += arm.GetPulls()
		}
		if pulls == 12 {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("policy B knows %d of 12 rewards; logs:\naggregator:\n%s\nB:\n%s", pulls, agg.Logs(), polB.Logs())
		case <-time.After(10 * time.Millisecond):
		}
	}

	for name, d := range map[string]*daemontest.Daemon{"policy A": polA, "policy B": polB, "feedback": fb, "aggregator": agg} {
		if err := d.Stop(); err != nil {
			t.Errorf("%s stopped with %v, want a clean shutdown", name, err)
		}
	}
}
