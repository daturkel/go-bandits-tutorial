// Package services_test wires the three services together, in one process,
// over in-memory connections: the same wiring as in production with the
// network taken out.
package services_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"google.golang.org/grpc"

	"banditlab/internal/bandit"
	pb "banditlab/internal/gen/bandit/v1"
	"banditlab/internal/rpcx/rpcxtest"
	"banditlab/internal/services/aggregator"
	"banditlab/internal/services/feedback"
	"banditlab/internal/services/policy"
	"banditlab/internal/store"
)

type system struct {
	policyA, policyB pb.PolicyServiceClient
	feedback         pb.FeedbackServiceClient
	store            store.Store
}

func newSystem(t *testing.T) *system {
	t.Helper()
	log := slog.New(slog.DiscardHandler)
	st := store.NewMemory(3)

	agg := aggregator.New(3, log)
	aggConn := rpcxtest.Start(t, func(s *grpc.Server) { pb.RegisterAggregatorServiceServer(s, agg) })
	aggClient := pb.NewAggregatorServiceClient(aggConn)

	fb := feedback.New(st, 3, feedback.NewReporter(aggClient, "test"), log)
	fbConn := rpcxtest.Start(t, func(s *grpc.Server) { pb.RegisterFeedbackServiceServer(s, fb) })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	sys := &system{feedback: pb.NewFeedbackServiceClient(fbConn), store: st}
	for _, dst := range []*pb.PolicyServiceClient{&sys.policyA, &sys.policyB} {
		pol, err := bandit.NewSnapshotPolicy("ucb1", 3, bandit.NewRNG(1, bandit.StreamPolicy))
		if err != nil {
			t.Fatal(err)
		}
		svc := policy.New(pol, st, time.Minute, log)
		conn := rpcxtest.Start(t, func(s *grpc.Server) { pb.RegisterPolicyServiceServer(s, svc) })
		*dst = pb.NewPolicyServiceClient(conn)
		go svc.Follow(ctx, aggClient, 10*time.Millisecond)
	}
	return sys
}

// waitPulls polls a policy service until arm has the given number of pulls.
func waitPulls(t *testing.T, c pb.PolicyServiceClient, arm int, want int64) *pb.StatsResponse {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		stats, err := c.Stats(context.Background(), &pb.StatsRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if stats.GetArms()[arm].GetPulls() == want {
			return stats
		}
		if time.Now().After(deadline) {
			t.Fatalf("arm %d has %d pulls after 5s, want %d", arm, stats.GetArms()[arm].GetPulls(), want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// A reward sent to the feedback service reaches a policy service that never
// saw the selection or the reward: through the store, the aggregator and its
// stream.
func TestRewardsReachEveryPolicyInstance(t *testing.T) {
	sys := newSystem(t)
	ctx := context.Background()

	for range 20 {
		sel, err := sys.policyA.Select(ctx, &pb.SelectRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := sys.feedback.Reward(ctx, &pb.RewardRequest{RequestId: sel.GetRequestId(), Reward: 1}); err != nil {
			t.Fatal(err)
		}
	}

	var total int64
	for _, c := range []pb.PolicyServiceClient{sys.policyA, sys.policyB} {
		deadline := time.Now().Add(5 * time.Second)
		for total = 0; total != 20; {
			stats, _ := c.Stats(ctx, &pb.StatsRequest{})
			total = 0
			for _, a := range stats.GetArms() {
				total += a.GetPulls()
			}
			if time.Now().After(deadline) {
				t.Fatalf("a policy instance knows %d of 20 rewards after 5s", total)
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func TestPendingSelectionsAreVisibleToOtherInstances(t *testing.T) {
	sys := newSystem(t)
	ctx := context.Background()
	sel, err := sys.policyA.Select(ctx, &pb.SelectRequest{})
	if err != nil {
		t.Fatal(err)
	}
	// Any change makes the aggregator publish, and each snapshot refreshes the
	// pending counts from the store. Reward a second selection to trigger one.
	other, _ := sys.policyA.Select(ctx, &pb.SelectRequest{})
	if _, err := sys.feedback.Reward(ctx, &pb.RewardRequest{RequestId: other.GetRequestId(), Reward: 0}); err != nil {
		t.Fatal(err)
	}
	waitPulls(t, sys.policyB, int(other.GetArm()), 1)
	stats, _ := sys.policyB.Stats(ctx, &pb.StatsRequest{})
	if got := stats.GetArms()[sel.GetArm()].GetPending(); got != 1 {
		t.Errorf("policyB sees %d pending on arm %d, want 1 (the selection made on policyA)", got, sel.GetArm())
	}
}
