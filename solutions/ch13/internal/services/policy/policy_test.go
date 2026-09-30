package policy_test

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"banditlab/internal/bandit"
	pb "banditlab/internal/gen/bandit/v1"
	"banditlab/internal/rpcx/rpcxtest"
	"banditlab/internal/services/policy"
	"banditlab/internal/store"
)

func newService(t *testing.T, st store.Store) (*policy.Service, pb.PolicyServiceClient) {
	t.Helper()
	pol, err := bandit.NewSnapshotPolicy("ucb1", 3, bandit.NewRNG(1, bandit.StreamPolicy))
	if err != nil {
		t.Fatal(err)
	}
	svc := policy.New(pol, st, time.Minute, slog.New(slog.DiscardHandler))
	conn := rpcxtest.Start(t, func(s *grpc.Server) { pb.RegisterPolicyServiceServer(s, svc) })
	return svc, pb.NewPolicyServiceClient(conn)
}

func TestSelectRecordsAPendingSelection(t *testing.T) {
	st := store.NewMemory(3)
	_, c := newService(t, st)
	resp, err := c.Select(context.Background(), &pb.SelectRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetRequestId() == "" || resp.GetArm() < 0 || resp.GetArm() > 2 {
		t.Fatalf("Select = %v", resp)
	}
	// The id must be usable by whoever handles the reward.
	if arm, err := st.Reward(context.Background(), resp.GetRequestId(), 1); err != nil || int32(arm) != resp.GetArm() {
		t.Errorf("Reward(%q) = %d, %v; want arm %d", resp.GetRequestId(), arm, err, resp.GetArm())
	}
}

func TestSelectionsAreSpreadWhileRewardsAreOutstanding(t *testing.T) {
	_, c := newService(t, store.NewMemory(3))
	var arms []int32
	for range 3 {
		resp, err := c.Select(context.Background(), &pb.SelectRequest{})
		if err != nil {
			t.Fatal(err)
		}
		arms = append(arms, resp.GetArm())
	}
	if arms[0] != 0 || arms[1] != 1 || arms[2] != 2 {
		t.Errorf("arms = %v, want [0 1 2]", arms)
	}
}

// brokenStore fails every AddPending.
type brokenStore struct{ store.Store }

func (brokenStore) AddPending(context.Context, string, int, time.Duration) error {
	return errors.New("connection refused")
}

func TestSelectFailsWithUnavailableAndReleasesTheArm(t *testing.T) {
	_, c := newService(t, brokenStore{store.NewMemory(3)})
	for range 4 {
		if _, err := c.Select(context.Background(), &pb.SelectRequest{}); status.Code(err) != codes.Unavailable {
			t.Fatalf("err = %v, want Unavailable", err)
		}
	}
	stats, _ := c.Stats(context.Background(), &pb.StatsRequest{})
	for _, a := range stats.GetArms() {
		if a.GetPending() != 0 {
			t.Errorf("arm %d still has %d pending after failed selections", a.GetArm(), a.GetPending())
		}
	}
}

func TestSelectHonoursACancelledContext(t *testing.T) {
	_, c := newService(t, store.NewMemory(3))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Select(ctx, &pb.SelectRequest{}); status.Code(err) != codes.Canceled {
		t.Errorf("err = %v, want Canceled", err)
	}
}

func TestApplyReplacesStateAndRecordsVersion(t *testing.T) {
	st := store.NewMemory(3)
	svc, c := newService(t, st)
	st.AddPending(context.Background(), "waiting", 1, time.Minute)

	totals := []bandit.ArmTotals{{Arm: 0, Pulls: 10, RewardSum: 2}, {Arm: 1, Pulls: 10, RewardSum: 9}, {Arm: 2, Pulls: 10, RewardSum: 5}}
	if err := svc.Apply(context.Background(), 7, totals); err != nil {
		t.Fatal(err)
	}
	stats, _ := c.Stats(context.Background(), &pb.StatsRequest{})
	if stats.GetSnapshotVersion() != 7 || stats.GetArms()[1].GetPulls() != 10 || stats.GetArms()[1].GetMean() != 0.9 {
		t.Errorf("stats = %v", stats)
	}
	if stats.GetArms()[1].GetPending() != 1 {
		t.Errorf("pending on arm 1 = %d, want 1 (from the store)", stats.GetArms()[1].GetPending())
	}
	if err := svc.Apply(context.Background(), 8, totals[:2]); !errors.Is(err, bandit.ErrBadTotals) {
		t.Errorf("Apply with two arms = %v, want ErrBadTotals", err)
	}
	if stats, _ := c.Stats(context.Background(), &pb.StatsRequest{}); stats.GetSnapshotVersion() != 7 {
		t.Error("a rejected snapshot must not change the version")
	}
}

// fakeAggregator serves Watch from a script: each entry is one connection's
// worth of snapshots, after which the stream ends with an error.
type fakeAggregator struct {
	pb.UnimplementedAggregatorServiceServer
	connections atomic.Int32
	script      [][]*pb.WatchResponse
}

func (f *fakeAggregator) Watch(_ *pb.WatchRequest, s pb.AggregatorService_WatchServer) error {
	n := int(f.connections.Add(1)) - 1
	if n >= len(f.script) {
		<-s.Context().Done()
		return s.Context().Err()
	}
	for _, snap := range f.script[n] {
		if err := s.Send(snap); err != nil {
			return err
		}
	}
	return status.Error(codes.Unavailable, "restarting")
}

func snap(v uint64, pulls int64) *pb.WatchResponse {
	return &pb.WatchResponse{Version: v, Totals: []*pb.ArmTotals{{Arm: 0, Pulls: pulls, RewardSum: 0}, {Arm: 1}, {Arm: 2}}}
}

func TestFollowAppliesSnapshotsAndReconnects(t *testing.T) {
	fake := &fakeAggregator{script: [][]*pb.WatchResponse{
		{snap(1, 5)},
		{snap(2, 9), {Version: 3, Totals: nil}, snap(4, 12)}, // version 3 is invalid and must be skipped
	}}
	conn := rpcxtest.Start(t, func(s *grpc.Server) { pb.RegisterAggregatorServiceServer(s, fake) })

	svc, c := newService(t, store.NewMemory(3))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.Follow(ctx, pb.NewAggregatorServiceClient(conn), 10*time.Millisecond)

	deadline := time.Now().Add(5 * time.Second)
	for {
		stats, _ := c.Stats(ctx, &pb.StatsRequest{})
		if stats.GetSnapshotVersion() == 4 {
			if stats.GetArms()[0].GetPulls() != 12 {
				t.Errorf("stats = %v", stats)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("still at snapshot %d after 5s (connections: %d)", stats.GetSnapshotVersion(), fake.connections.Load())
		}
		time.Sleep(5 * time.Millisecond)
	}
	if fake.connections.Load() < 2 {
		t.Errorf("connections = %d, want a reconnect", fake.connections.Load())
	}
}
