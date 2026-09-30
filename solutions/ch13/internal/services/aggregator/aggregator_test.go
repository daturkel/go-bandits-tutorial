package aggregator_test

import (
	"context"
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
	"banditlab/internal/services/aggregator"
	"banditlab/internal/store"
)

func start(t *testing.T, arms int) (*aggregator.Service, pb.AggregatorServiceClient) {
	t.Helper()
	svc := aggregator.New(arms, slog.New(slog.DiscardHandler))
	conn := rpcxtest.Start(t, func(s *grpc.Server) { pb.RegisterAggregatorServiceServer(s, svc) })
	return svc, pb.NewAggregatorServiceClient(conn)
}

func delta(arm int, pulls int64, sum float64, arms int) []*pb.ArmTotals {
	d := make([]*pb.ArmTotals, arms)
	for i := range d {
		d[i] = &pb.ArmTotals{Arm: int32(i)}
	}
	d[arm].Pulls, d[arm].RewardSum = pulls, sum
	return d
}

func recv(t *testing.T, s pb.AggregatorService_WatchClient) *pb.WatchResponse {
	t.Helper()
	type result struct {
		r   *pb.WatchResponse
		err error
	}
	ch := make(chan result, 1)
	go func() { r, err := s.Recv(); ch <- result{r, err} }()
	select {
	case res := <-ch:
		if res.err != nil {
			t.Fatalf("Recv: %v", res.err)
		}
		return res.r
	case <-time.After(3 * time.Second):
		t.Fatal("no snapshot within 3s")
		return nil
	}
}

func TestReportMergesDeltas(t *testing.T) {
	_, c := start(t, 3)
	ctx := context.Background()
	c.Report(ctx, &pb.ReportRequest{Source: "a", Deltas: delta(0, 4, 1, 3)})
	resp, err := c.Report(ctx, &pb.ReportRequest{Source: "b", Deltas: delta(0, 6, 5, 3)})
	if err != nil || resp.GetVersion() != 2 {
		t.Fatalf("Report = %v, %v; want version 2", resp, err)
	}
	w, err := c.Watch(ctx, &pb.WatchRequest{})
	if err != nil {
		t.Fatal(err)
	}
	snap := recv(t, w)
	if got := snap.GetTotals()[0]; got.GetPulls() != 10 || got.GetRewardSum() != 6 {
		t.Errorf("arm 0 = %v, want 10 pulls and reward sum 6", got)
	}
}

func TestReportRejectsBadDeltas(t *testing.T) {
	_, c := start(t, 3)
	for name, req := range map[string]*pb.ReportRequest{
		"wrong number of arms": {Deltas: delta(0, 1, 1, 2)},
		"negative pulls":       {Deltas: delta(0, -1, 0, 3)},
		"reward above pulls":   {Deltas: delta(1, 2, 3, 3)},
		"nothing":              {},
	} {
		_, err := c.Report(context.Background(), req)
		if status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: err = %v, want InvalidArgument", name, err)
		}
	}
}

func TestWatchSendsCurrentStateThenChanges(t *testing.T) {
	_, c := start(t, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w, err := c.Watch(ctx, &pb.WatchRequest{})
	if err != nil {
		t.Fatal(err)
	}
	first := recv(t, w)
	if first.GetVersion() != 0 || len(first.GetTotals()) != 2 {
		t.Errorf("first snapshot = %v, want version 0 with 2 arms", first)
	}
	c.Report(ctx, &pb.ReportRequest{Deltas: delta(1, 3, 2, 2)})
	next := recv(t, w)
	if next.GetVersion() != 1 || next.GetTotals()[1].GetPulls() != 3 {
		t.Errorf("second snapshot = %v", next)
	}
}

func TestEveryWatcherSeesTheLatestState(t *testing.T) {
	_, c := start(t, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const watchers = 5
	var streams []pb.AggregatorService_WatchClient
	for range watchers {
		w, err := c.Watch(ctx, &pb.WatchRequest{})
		if err != nil {
			t.Fatal(err)
		}
		recv(t, w) // the initial snapshot
		streams = append(streams, w)
	}
	c.Report(ctx, &pb.ReportRequest{Deltas: delta(0, 1, 1, 2)})
	for i, w := range streams {
		if snap := recv(t, w); snap.GetTotals()[0].GetPulls() != 1 {
			t.Errorf("watcher %d saw %v", i, snap)
		}
	}
}

// Many sources report at once; the totals must add up exactly (run with -race).
func TestConcurrentReports(t *testing.T) {
	_, c := start(t, 2)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				if _, err := c.Report(context.Background(), &pb.ReportRequest{Deltas: delta(0, 1, 1, 2)}); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	wg.Wait()
	w, _ := c.Watch(context.Background(), &pb.WatchRequest{})
	snap := recv(t, w)
	if snap.GetVersion() != 400 || snap.GetTotals()[0].GetPulls() != 400 {
		t.Errorf("snapshot = version %d, %d pulls; want 400 and 400", snap.GetVersion(), snap.GetTotals()[0].GetPulls())
	}
}

func TestWatchEndsWhenTheClientLeaves(t *testing.T) {
	_, c := start(t, 2)
	ctx, cancel := context.WithCancel(context.Background())
	w, _ := c.Watch(ctx, &pb.WatchRequest{})
	recv(t, w)
	cancel()
	if _, err := w.Recv(); status.Code(err) != codes.Canceled {
		t.Errorf("Recv after cancel = %v, want Canceled", err)
	}
}

func TestReconcileCorrectsDrift(t *testing.T) {
	svc, c := start(t, 2)
	st := store.NewMemory(2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// A reward reached the store but its report to the aggregator was lost.
	st.AddPending(ctx, "x", 1, time.Minute)
	if _, err := st.Reward(ctx, "x", 1); err != nil {
		t.Fatal(err)
	}
	go svc.Reconcile(ctx, st, 20*time.Millisecond)

	w, _ := c.Watch(ctx, &pb.WatchRequest{})
	for {
		snap := recv(t, w)
		if snap.GetTotals()[1].GetPulls() == 1 {
			return
		}
	}
}

func TestResetValidates(t *testing.T) {
	svc, _ := start(t, 2)
	if err := svc.Reset([]bandit.ArmTotals{{Arm: 0}}); err == nil {
		t.Error("Reset with the wrong number of arms should fail")
	}
}

func TestCloseEndsWatchStreamsWithUnavailable(t *testing.T) {
	svc, c := start(t, 2)
	w, _ := c.Watch(context.Background(), &pb.WatchRequest{})
	recv(t, w)
	svc.Close()
	svc.Close() // closing twice is harmless
	if _, err := w.Recv(); status.Code(err) != codes.Unavailable {
		t.Errorf("Recv after Close = %v, want Unavailable", err)
	}
}
