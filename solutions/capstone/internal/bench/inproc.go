package bench

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	pb "banditlab/internal/gen/bandit/v1"
)

// inproc is a gRPC server reachable through an in-memory pipe. Calls go through
// the full gRPC stack (serialisation, interceptors, streaming) but not through
// a network card, so a benchmark measures the services and not the loopback
// device.
type inproc struct {
	lis *bufconn.Listener
	srv *grpc.Server
}

func newInproc(register func(*grpc.Server)) *inproc {
	p := &inproc{lis: bufconn.Listen(1 << 20), srv: grpc.NewServer()}
	register(p.srv)
	go p.srv.Serve(p.lis)
	return p
}

// dial returns a client connection to the server, with extra options applied.
func (p *inproc) dial(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append(opts,
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return p.lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	return grpc.NewClient("passthrough:///inproc", opts...)
}

// stop closes the server and its connections at once.
func (p *inproc) stop() { p.srv.Stop() }

// laggedClient is an aggregator client whose Watch stream delivers every
// snapshot lag after the aggregator sent it. It stands in for the time a
// state transfer takes: a slow link, a busy replica, a large snapshot.
type laggedClient struct {
	pb.AggregatorServiceClient
	lag time.Duration
}

func (c laggedClient) Watch(ctx context.Context, in *pb.WatchRequest, opts ...grpc.CallOption) (pb.AggregatorService_WatchClient, error) {
	inner, err := c.AggregatorServiceClient.Watch(ctx, in, opts...)
	if err != nil || c.lag <= 0 {
		return inner, err
	}
	s := &lagStream{ServerStreamingClient: inner, ctx: ctx, lag: c.lag, queue: make(chan delayed, 1<<12)}
	go s.pump()
	return s, nil
}

type delayed struct {
	msg *pb.WatchResponse
	err error
	at  time.Time // when the receiver may see it
}

// lagStream reads the real stream as fast as messages come and hands each one
// out when it is lag old. Delaying each message independently matters: if the
// receiver slept lag after every message instead, a burst of n snapshots would
// take n*lag to deliver, and the lag would grow without bound.
type lagStream struct {
	grpc.ServerStreamingClient[pb.WatchResponse]
	ctx   context.Context
	lag   time.Duration
	queue chan delayed
}

func (s *lagStream) pump() {
	for {
		msg, err := s.ServerStreamingClient.Recv()
		select {
		case s.queue <- delayed{msg, err, time.Now().Add(s.lag)}:
		case <-s.ctx.Done():
			return
		}
		if err != nil {
			return
		}
	}
}

func (s *lagStream) Recv() (*pb.WatchResponse, error) {
	select {
	case d := <-s.queue:
		if !sleep(s.ctx, time.Until(d.at)) {
			return nil, s.ctx.Err()
		}
		return d.msg, d.err
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}
