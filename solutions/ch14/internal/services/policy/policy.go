// Package policy serves arm selections over gRPC and follows the
// aggregator's snapshots to stay current.
package policy

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/status"

	"banditlab/internal/bandit"
	pb "banditlab/internal/gen/bandit/v1"
	"banditlab/internal/rpcx"
	"banditlab/internal/store"
	"banditlab/internal/wire"
)

// Service implements pb.PolicyServiceServer.
type Service struct {
	pb.UnimplementedPolicyServiceServer

	policy  *bandit.Locked
	store   store.Store
	ttl     time.Duration
	log     *slog.Logger
	version atomic.Uint64 // the last aggregator snapshot applied

	instance string // names this instance in responses
}

// Option customises a Service.
type Option func(*Service)

// WithInstance names this instance. Responses carry the name, so a caller
// behind a load balancer can see which replica answered.
func WithInstance(name string) Option {
	return func(s *Service) { s.instance = name }
}

// New returns a Service. Selections are remembered in st for ttl.
func New(pol bandit.SnapshotPolicy, st store.Store, ttl time.Duration, log *slog.Logger, opts ...Option) *Service {
	s := &Service{policy: bandit.NewLocked(pol), store: st, ttl: ttl, log: log}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Select chooses an arm and records the selection so a reward can find it.
func (s *Service) Select(ctx context.Context, _ *pb.SelectRequest) (*pb.SelectResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	arm := s.policy.Select()
	id := rand.Text()
	if err := s.store.AddPending(ctx, id, arm, s.ttl); err != nil {
		s.policy.Abandon(arm)
		return nil, rpcx.FromStore(err)
	}
	return &pb.SelectResponse{RequestId: id, Arm: int32(arm), Instance: s.instance}, nil
}

// Stats reports the policy's beliefs and which snapshot they came from.
func (s *Service) Stats(context.Context, *pb.StatsRequest) (*pb.StatsResponse, error) {
	return &pb.StatsResponse{
		Instance:        s.instance,
		Policy:          s.policy.Name(),
		SnapshotVersion: s.version.Load(),
		Arms:            wire.StatsToProto(s.policy.Snapshot()),
	}, nil
}

// Apply replaces the policy's state with a snapshot of the merged totals and
// refreshes the pending counts from the store.
func (s *Service) Apply(ctx context.Context, version uint64, totals []bandit.ArmTotals) error {
	if err := s.policy.Restore(totals); err != nil {
		return err
	}
	pending, err := s.store.PendingCounts(ctx)
	if err != nil {
		return err
	}
	if err := s.policy.SetPending(pending); err != nil {
		return err
	}
	s.version.Store(version)
	return nil
}

// Follow subscribes to the aggregator and applies every snapshot it sends,
// until ctx ends. If the stream breaks (the aggregator restarted, the network
// dropped) it waits and subscribes again; the service keeps answering Select
// from its last snapshot in the meantime.
func (s *Service) Follow(ctx context.Context, client pb.AggregatorServiceClient, retry time.Duration) {
	for ctx.Err() == nil {
		err := s.follow(ctx, client)
		if ctx.Err() != nil {
			return
		}
		s.log.Warn("lost the aggregator, will retry", "err", err, "retry_in", retry)
		select {
		case <-time.After(retry):
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) follow(ctx context.Context, client pb.AggregatorServiceClient) error {
	stream, err := client.Watch(ctx, &pb.WatchRequest{})
	if err != nil {
		return err
	}
	for {
		snap, err := stream.Recv()
		if err != nil {
			return err
		}
		if err := s.Apply(ctx, snap.GetVersion(), wire.TotalsFromProto(snap.GetTotals())); err != nil {
			// A bad snapshot is a bug on the other side; keep the old state
			// and wait for the next one instead of dropping the stream.
			if errors.Is(err, bandit.ErrBadTotals) {
				s.log.Error("ignoring an invalid snapshot", "version", snap.GetVersion(), "err", err)
				continue
			}
			s.log.Warn("could not apply snapshot", "version", snap.GetVersion(), "err", err)
		}
	}
}
