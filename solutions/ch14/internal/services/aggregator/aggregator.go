// Package aggregator keeps the merged per-arm counts of the whole system and
// tells everyone who is interested when they change.
package aggregator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"banditlab/internal/bandit"
	pb "banditlab/internal/gen/bandit/v1"
	"banditlab/internal/store"
	"banditlab/internal/wire"
)

// Service implements pb.AggregatorServiceServer.
type Service struct {
	pb.UnimplementedAggregatorServiceServer // methods we do not write return Unimplemented

	log  *slog.Logger
	arms int // fixed at construction, so it can be read without the lock

	mu      sync.Mutex
	totals  []bandit.ArmTotals
	version uint64
	changed chan struct{} // closed, and replaced, whenever totals change

	done     chan struct{} // closed by Close
	closeOne sync.Once
}

// New returns a Service for nArms arms, starting from zero counts.
func New(nArms int, log *slog.Logger) *Service {
	totals := make([]bandit.ArmTotals, nArms)
	for i := range totals {
		totals[i].Arm = i
	}
	return &Service{log: log, arms: nArms, totals: totals, changed: make(chan struct{}), done: make(chan struct{})}
}

// Close ends every Watch stream, with Unavailable so that clients know to
// reconnect. Call it when the server starts to shut down: a graceful stop
// waits for streams like any other call, and a stream that only ends when the
// client leaves would use up the whole grace period.
func (s *Service) Close() { s.closeOne.Do(func() { close(s.done) }) }

// Report merges one source's deltas into the totals.
func (s *Service) Report(ctx context.Context, req *pb.ReportRequest) (*pb.ReportResponse, error) {
	deltas := wire.TotalsFromProto(req.GetDeltas())
	if err := validate(deltas, s.arms); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	merged, err := bandit.Merge(s.totals, deltas)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	s.set(merged)
	return &pb.ReportResponse{Version: s.version}, nil
}

// Reset replaces the totals, for example with what the database holds. Use
// it to correct drift: a lost Report leaves the aggregator behind the store.
func (s *Service) Reset(totals []bandit.ArmTotals) error {
	if err := validate(totals, s.arms); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.set(append([]bandit.ArmTotals(nil), totals...))
	return nil
}

// set installs new totals and wakes every watcher. The caller holds s.mu.
func (s *Service) set(totals []bandit.ArmTotals) {
	s.totals = totals
	s.version++
	close(s.changed)                // everyone waiting on the old channel wakes up
	s.changed = make(chan struct{}) // and the next change closes this one
}

// Watch sends the current totals, then a new snapshot after each change.
func (s *Service) Watch(_ *pb.WatchRequest, stream pb.AggregatorService_WatchServer) error {
	ctx := stream.Context()
	var sent uint64
	sentAny := false
	for {
		s.mu.Lock()
		snap := &pb.WatchResponse{Version: s.version, Totals: wire.TotalsToProto(s.totals)}
		changed := s.changed
		s.mu.Unlock()

		// A watcher that is slower than the writers skips the versions in
		// between: each message is the full state, so nothing is lost.
		if !sentAny || snap.Version != sent {
			if err := stream.Send(snap); err != nil {
				return err
			}
			sent, sentAny = snap.Version, true
		}
		select {
		case <-changed:
		case <-s.done:
			return status.Error(codes.Unavailable, "aggregator is shutting down")
		case <-ctx.Done():
			return status.FromContextError(ctx.Err()).Err()
		}
	}
}

// Reconcile resets the totals from the store every interval, until ctx ends.
// Reports can be lost (a feedback service that could not reach us, a restart
// of ours), and the store, which every reward passes through first, is the
// truth. Reconciling turns "lost update" from a permanent error into a
// delay of at most one interval.
func (s *Service) Reconcile(ctx context.Context, st store.Store, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := s.reconcileOnce(ctx, st); err != nil && ctx.Err() == nil {
			s.log.Warn("reconcile failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) reconcileOnce(ctx context.Context, st store.Store) error {
	totals, err := st.Totals(ctx)
	if err != nil {
		return err
	}
	// Only publish if something differs, so idle systems stay quiet.
	s.mu.Lock()
	same := equal(totals, s.totals)
	s.mu.Unlock()
	if same {
		return nil
	}
	return s.Reset(totals)
}

// ExpireLoop removes selections whose time to live has passed, every
// interval, until ctx ends. If implicit is non-nil, each selection that
// expired without a reward is counted as if it had received that reward, and
// the aggregator adds those counts itself. It does this because it is the one
// place that owns the merged totals: the store applies the same counts to its
// own totals, and both then agree.
func (s *Service) ExpireLoop(ctx context.Context, st store.Store, interval time.Duration, implicit *float64) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		expired, err := st.Expire(ctx, implicit)
		if err != nil {
			if ctx.Err() == nil {
				s.log.Warn("expire failed", "err", err)
			}
			continue
		}
		if len(expired) == 0 {
			continue
		}
		total := 0
		for _, n := range expired {
			total += n
		}
		if implicit != nil {
			delta := make([]bandit.ArmTotals, s.arms)
			for i := range delta {
				delta[i].Arm = i
			}
			for arm, n := range expired {
				delta[arm].Pulls = int64(n)
				delta[arm].RewardSum = float64(n) * *implicit
			}
			s.mu.Lock()
			merged, err := bandit.Merge(s.totals, delta)
			if err == nil {
				s.set(merged)
			}
			s.mu.Unlock()
		}
		s.log.Info("expired selections", "count", total, "counted_as", implicit)
	}
}

func equal(a, b []bandit.ArmTotals) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// validate checks a message that came from outside the process. A protobuf
// message can hold any values, so nothing about it can be assumed.
func validate(t []bandit.ArmTotals, nArms int) error {
	if len(t) != nArms {
		return fmt.Errorf("got %d arms, want %d", len(t), nArms)
	}
	for i, a := range t {
		switch {
		case a.Arm != i:
			return fmt.Errorf("entry %d is for arm %d", i, a.Arm)
		case a.Pulls < 0:
			return fmt.Errorf("arm %d has %d pulls", i, a.Pulls)
		case !(a.RewardSum >= 0 && a.RewardSum <= float64(a.Pulls)+1e-9):
			return fmt.Errorf("arm %d has reward sum %v for %d pulls", i, a.RewardSum, a.Pulls)
		}
	}
	return nil
}
