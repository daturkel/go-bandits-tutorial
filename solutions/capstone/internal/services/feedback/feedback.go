// Package feedback receives rewards, records them, and passes the news on.
package feedback

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"banditlab/internal/bandit"
	pb "banditlab/internal/gen/bandit/v1"
	"banditlab/internal/rpcx"
	"banditlab/internal/store"
	"banditlab/internal/wire"
)

const reportTimeout = 2 * time.Second

// Reporter delivers counts to whoever merges them. The gRPC client of the
// aggregator implements it (see NewReporter); tests use a fake.
type Reporter interface {
	Report(ctx context.Context, deltas []bandit.ArmTotals) error
}

// Service implements pb.FeedbackServiceServer.
type Service struct {
	pb.UnimplementedFeedbackServiceServer

	store    store.Store
	reporter Reporter // nil: nobody to tell
	arms     int
	log      *slog.Logger
}

// New returns a Service. reporter may be nil.
func New(st store.Store, nArms int, reporter Reporter, log *slog.Logger) *Service {
	return &Service{store: st, arms: nArms, reporter: reporter, log: log}
}

// Reward records one reward.
func (s *Service) Reward(ctx context.Context, req *pb.RewardRequest) (*pb.RewardResponse, error) {
	if err := check(req.GetRequestId(), req.GetReward()); err != nil {
		return nil, err
	}
	arm, err := s.store.Reward(ctx, req.GetRequestId(), req.GetReward())
	if err != nil {
		return nil, rpcx.FromStore(err)
	}
	s.tell(ctx, s.delta(arm, req.GetReward()))
	return &pb.RewardResponse{Arm: int32(arm)}, nil
}

// RewardBatch records every reward on the stream and answers once at the end.
// Unknown, expired and repeated ids are counted as rejected and do not stop
// the batch; a failing store does.
func (s *Service) RewardBatch(stream pb.FeedbackService_RewardBatchServer) error {
	ctx := stream.Context()
	var accepted, rejected int32
	total := s.zero()
	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break // the client is done sending
		}
		if err != nil {
			return err
		}
		if err := check(req.GetRequestId(), req.GetReward()); err != nil {
			rejected++
			continue
		}
		arm, err := s.store.Reward(ctx, req.GetRequestId(), req.GetReward())
		switch {
		case err == nil:
			accepted++
			total[arm].Pulls++
			total[arm].RewardSum += req.GetReward()
		case errors.Is(err, store.ErrUnknownID), errors.Is(err, store.ErrAlreadyRewarded):
			rejected++
		default:
			return rpcx.FromStore(err)
		}
	}
	if accepted > 0 {
		s.tell(ctx, total) // one report for the whole batch
	}
	return stream.SendAndClose(&pb.RewardBatchResponse{Accepted: accepted, Rejected: rejected})
}

func check(id string, reward float64) error {
	switch {
	case id == "":
		return status.Error(codes.InvalidArgument, "request_id is required")
	case !(reward >= 0 && reward <= 1): // written this way so NaN fails too
		return status.Error(codes.InvalidArgument, "reward must be between 0 and 1")
	}
	return nil
}

// tell passes deltas to the reporter. A failure is logged and otherwise
// ignored: the reward is already in the store, which is the truth, and the
// aggregator reconciles against it. Failing the caller here would make them
// retry a reward that was in fact recorded.
func (s *Service) tell(ctx context.Context, deltas []bandit.ArmTotals) {
	if s.reporter == nil {
		return
	}
	// The caller's deadline may be nearly used up; the report should not
	// depend on it, but it should not hang either.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), reportTimeout)
	defer cancel()
	if err := s.reporter.Report(ctx, deltas); err != nil {
		s.log.Warn("could not report to the aggregator; it will catch up when it reconciles", "err", err)
	}
}

func (s *Service) zero() []bandit.ArmTotals {
	z := make([]bandit.ArmTotals, s.arms)
	for i := range z {
		z[i].Arm = i
	}
	return z
}

func (s *Service) delta(arm int, reward float64) []bandit.ArmTotals {
	d := s.zero()
	d[arm].Pulls = 1
	d[arm].RewardSum = reward
	return d
}

// NewReporter returns a Reporter that sends deltas to an aggregator over
// gRPC, naming this service as the source.
func NewReporter(client pb.AggregatorServiceClient, source string) Reporter {
	return &grpcReporter{client: client, source: source}
}

type grpcReporter struct {
	client pb.AggregatorServiceClient
	source string
}

func (r *grpcReporter) Report(ctx context.Context, deltas []bandit.ArmTotals) error {
	_, err := r.client.Report(ctx, &pb.ReportRequest{Source: r.source, Deltas: wire.TotalsToProto(deltas)})
	return err
}
