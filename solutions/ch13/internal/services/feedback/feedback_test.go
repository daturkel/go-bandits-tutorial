package feedback_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"banditlab/internal/bandit"
	pb "banditlab/internal/gen/bandit/v1"
	"banditlab/internal/rpcx/rpcxtest"
	"banditlab/internal/services/feedback"
	"banditlab/internal/store"
)

// recorder is a Reporter that remembers what it was told.
type recorder struct {
	mu   sync.Mutex
	got  [][]bandit.ArmTotals
	fail error
}

func (r *recorder) Report(_ context.Context, d []bandit.ArmTotals) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, d)
	return r.fail
}

func setup(t *testing.T, rep feedback.Reporter) (pb.FeedbackServiceClient, store.Store) {
	t.Helper()
	st := store.NewMemory(3)
	svc := feedback.New(st, 3, rep, slog.New(slog.DiscardHandler))
	conn := rpcxtest.Start(t, func(s *grpc.Server) { pb.RegisterFeedbackServiceServer(s, svc) })
	return pb.NewFeedbackServiceClient(conn), st
}

func pending(t *testing.T, st store.Store, id string, arm int) {
	t.Helper()
	if err := st.AddPending(context.Background(), id, arm, time.Minute); err != nil {
		t.Fatal(err)
	}
}

func TestRewardStatusCodes(t *testing.T) {
	c, st := setup(t, nil)
	pending(t, st, "a", 2)
	ctx := context.Background()

	resp, err := c.Reward(ctx, &pb.RewardRequest{RequestId: "a", Reward: 1})
	if err != nil || resp.GetArm() != 2 {
		t.Fatalf("Reward = %v, %v; want arm 2", resp, err)
	}
	tests := []struct {
		name string
		req  *pb.RewardRequest
		want codes.Code
	}{
		{"repeat", &pb.RewardRequest{RequestId: "a", Reward: 1}, codes.AlreadyExists},
		{"unknown", &pb.RewardRequest{RequestId: "zzz", Reward: 1}, codes.NotFound},
		{"no id", &pb.RewardRequest{Reward: 1}, codes.InvalidArgument},
		{"reward too big", &pb.RewardRequest{RequestId: "b", Reward: 1.5}, codes.InvalidArgument},
	}
	for _, tc := range tests {
		if _, err := c.Reward(ctx, tc.req); status.Code(err) != tc.want {
			t.Errorf("%s: err = %v, want %v", tc.name, err, tc.want)
		}
	}
}

func TestRewardIsReported(t *testing.T) {
	rec := &recorder{}
	c, st := setup(t, rec)
	pending(t, st, "a", 1)
	if _, err := c.Reward(context.Background(), &pb.RewardRequest{RequestId: "a", Reward: 0.5}); err != nil {
		t.Fatal(err)
	}
	if len(rec.got) != 1 || rec.got[0][1].Pulls != 1 || rec.got[0][1].RewardSum != 0.5 || rec.got[0][0].Pulls != 0 {
		t.Errorf("reported %+v, want one delta with a single pull on arm 1", rec.got)
	}
}

// The store has the reward, so the caller must not be told to retry it.
func TestReporterFailureDoesNotFailTheCall(t *testing.T) {
	rec := &recorder{fail: errors.New("aggregator down")}
	c, st := setup(t, rec)
	pending(t, st, "a", 0)
	if _, err := c.Reward(context.Background(), &pb.RewardRequest{RequestId: "a", Reward: 1}); err != nil {
		t.Errorf("Reward = %v, want success despite the failed report", err)
	}
	totals, _ := st.Totals(context.Background())
	if totals[0].Pulls != 1 {
		t.Error("the reward should be in the store")
	}
}

func TestRewardBatch(t *testing.T) {
	rec := &recorder{}
	c, st := setup(t, rec)
	for i, id := range []string{"a", "b", "c", "d"} {
		pending(t, st, id, i%3)
	}
	stream, err := c.RewardBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []*pb.RewardBatchRequest{
		{RequestId: "a", Reward: 1},
		{RequestId: "b", Reward: 0},
		{RequestId: "a", Reward: 1},    // repeated
		{RequestId: "nope", Reward: 1}, // unknown
		{RequestId: "c", Reward: 7},    // invalid
		{RequestId: "d", Reward: 1},
	} {
		if err := stream.Send(r); err != nil {
			t.Fatal(err)
		}
	}
	sum, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatal(err)
	}
	if sum.GetAccepted() != 3 || sum.GetRejected() != 3 {
		t.Errorf("summary = %v, want 3 accepted and 3 rejected", sum)
	}
	if len(rec.got) != 1 {
		t.Fatalf("batch made %d reports, want 1", len(rec.got))
	}
	var pulls int64
	var reward float64
	for _, a := range rec.got[0] {
		pulls += a.Pulls
		reward += a.RewardSum
	}
	if pulls != 3 || reward != 2 {
		t.Errorf("reported %d pulls and reward %v, want 3 and 2", pulls, reward)
	}
}

func TestEmptyBatchReportsNothing(t *testing.T) {
	rec := &recorder{}
	c, _ := setup(t, rec)
	stream, _ := c.RewardBatch(context.Background())
	sum, err := stream.CloseAndRecv()
	if err != nil || sum.GetAccepted() != 0 || len(rec.got) != 0 {
		t.Errorf("empty batch: %v, %v, %d reports", sum, err, len(rec.got))
	}
}
