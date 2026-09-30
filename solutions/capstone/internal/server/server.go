// Package server exposes a bandit policy over HTTP.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"banditlab/internal/bandit"
	"banditlab/internal/metrics"
	"banditlab/internal/store"
)

// Defaults for options.
const (
	defaultPendingTTL = 10 * time.Minute
	// storeTimeout bounds every call to the store made while handling a
	// request, so a stuck database cannot hold requests open forever.
	storeTimeout = 3 * time.Second
)

// Server serves select and reward requests for one policy. The policy lives
// in memory for speed; the store holds what must survive a restart.
type Server struct {
	policy      *bandit.Locked
	store       store.Store
	log         *slog.Logger
	selectDelay time.Duration
	pendingTTL  time.Duration
	slowRequest time.Duration
	replicaID   string
	metrics     *metrics.Metrics
	draining    atomic.Bool
}

// Option adjusts a Server at construction.
type Option func(*Server)

// WithSelectDelay makes every selection take at least d, simulating a policy
// that is slow to compute (a model call, a feature lookup). Useful for
// exercising timeouts, shutdown and load behaviour.
func WithSelectDelay(d time.Duration) Option {
	return func(s *Server) { s.selectDelay = d }
}

// WithReplicaID names this instance in /stats, so that when several run
// behind a load balancer you can tell which one answered.
func WithReplicaID(id string) Option {
	return func(s *Server) { s.replicaID = id }
}

// WithSlowRequestThreshold sets how long a request may take before it is
// logged at Warn instead of Debug. The default is half a second.
func WithSlowRequestThreshold(d time.Duration) Option {
	return func(s *Server) { s.slowRequest = d }
}

// WithMetrics makes the server record request and business metrics, and serve
// them at GET /metrics. Without it, a server keeps no metrics.
func WithMetrics(m *metrics.Metrics) Option {
	return func(s *Server) {
		s.metrics = m
		m.TrackArms(s.policy.Snapshot)
	}
}

// WithPendingTTL sets how long a selection waits for its reward before it
// expires. Rewards arriving later are rejected as unknown.
func WithPendingTTL(d time.Duration) Option {
	return func(s *Server) { s.pendingTTL = d }
}

// New returns a Server driving policy and recording state in st. The policy
// must not be used elsewhere afterwards: the server wraps it in a lock. Call
// Restore before serving to load what st already knows.
func New(policy bandit.SnapshotPolicy, st store.Store, log *slog.Logger, opts ...Option) *Server {
	s := &Server{
		policy:      bandit.NewLocked(policy),
		store:       st,
		log:         log,
		pendingTTL:  defaultPendingTTL,
		slowRequest: 500 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SetDraining marks the server as shutting down: /healthz starts reporting
// 503, so a load balancer stops sending new requests while existing ones
// finish.
func (s *Server) SetDraining() { s.draining.Store(true) }

// Handler returns the routes wrapped in the standard middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /select", s.handleSelect)
	mux.HandleFunc("POST /reward", s.handleReward)
	mux.HandleFunc("GET /stats", s.handleStats)
	mux.HandleFunc("GET /healthz", s.handleHealth)

	// Outermost first: every request gets an id, the id is available to the
	// logger, and both the logger and the metrics see the 500 that
	// recoverPanic writes.
	mws := []Middleware{requestID, logging(s.log, s.slowRequest)}
	if s.metrics != nil {
		mux.Handle("GET /metrics", s.metrics.Handler())
		mws = append(mws, s.metrics.Middleware)
	}
	mws = append(mws, recoverPanic(s.log))
	return Chain(mux, mws...)
}

type selectResponse struct {
	RequestID string `json:"request_id"`
	Arm       int    `json:"arm"`
}

// handleSelect picks an arm and remembers it under this request's id, so the
// reward can be matched to it later.
func (s *Server) handleSelect(w http.ResponseWriter, r *http.Request) {
	id := RequestID(r.Context())
	if s.selectDelay > 0 {
		select {
		case <-time.After(s.selectDelay):
		case <-r.Context().Done():
			// The client hung up. Nobody will read a response; 499 is the
			// convention (from nginx) for "client closed request", and it
			// keeps the log honest.
			w.WriteHeader(499)
			return
		}
	}
	arm := s.policy.Select()

	ctx, cancel := context.WithTimeout(r.Context(), storeTimeout)
	defer cancel()
	if err := s.store.AddPending(ctx, id, arm, s.pendingTTL); err != nil {
		s.policy.Abandon(arm) // this selection will never get a reward: release it
		s.storeFailure(w, r, "record selection", err)
		return
	}
	if s.metrics != nil {
		s.metrics.Selected(arm)
	}
	writeJSON(w, http.StatusOK, selectResponse{RequestID: id, Arm: arm})
}

// rewardRequest uses a pointer for Reward so a missing field (nil) can be told
// apart from an explicit 0.
type rewardRequest struct {
	RequestID string   `json:"request_id"`
	Reward    *float64 `json:"reward"`
}

// handleReward records the outcome of an earlier selection.
func (s *Server) handleReward(w http.ResponseWriter, r *http.Request) {
	var req rewardRequest
	if err := decodeJSON(w, r, &req); err != nil {
		badBody(w, err)
		return
	}
	switch {
	case req.RequestID == "":
		writeError(w, http.StatusBadRequest, "request_id is required")
		return
	case req.Reward == nil:
		writeError(w, http.StatusBadRequest, "reward is required")
		return
	case !(*req.Reward >= 0 && *req.Reward <= 1):
		writeError(w, http.StatusUnprocessableEntity, "reward must be between 0 and 1")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), storeTimeout)
	defer cancel()
	arm, err := s.store.Reward(ctx, req.RequestID, *req.Reward)
	switch {
	case err == nil:
	case errors.Is(err, store.ErrUnknownID):
		writeError(w, http.StatusNotFound, err.Error())
		return
	case errors.Is(err, store.ErrAlreadyRewarded):
		writeError(w, http.StatusConflict, err.Error())
		return
	default:
		s.storeFailure(w, r, "record reward", err)
		return
	}

	// The reward is durable now. Teach the in-memory policy too. If the
	// process dies before this line, Restore rebuilds the same state.
	if s.metrics != nil {
		s.metrics.Rewarded(arm, *req.Reward)
	}
	s.policy.Update(arm, *req.Reward)
	w.WriteHeader(http.StatusNoContent)
}

type statsResponse struct {
	Replica string           `json:"replica,omitempty"`
	Policy  string           `json:"policy"`
	Selects int64            `json:"selects"`
	Updates int64            `json:"updates"`
	Arms    []bandit.ArmStat `json:"arms"`
}

// handleStats reports what the policy currently believes.
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	selects, updates := s.policy.Calls()
	writeJSON(w, http.StatusOK, statsResponse{
		Replica: s.replicaID,
		Policy:  s.policy.Name(),
		Selects: selects,
		Updates: updates,
		Arms:    s.policy.Snapshot(),
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if s.draining.Load() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "draining"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		s.log.Warn("health check: store unreachable", "id", RequestID(r.Context()), "err", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// storeFailure reports a store error to the client as 503 without exposing
// its details, and logs the details.
func (s *Server) storeFailure(w http.ResponseWriter, r *http.Request, what string, err error) {
	if s.metrics != nil {
		s.metrics.StoreError()
	}
	s.log.Error("store error", "id", RequestID(r.Context()), "operation", what, "err", err)
	writeError(w, http.StatusServiceUnavailable, "storage unavailable, try again")
}

// Sync makes the in-memory policy agree with the store: the totals every
// replica has written, and the selections every replica is waiting on. The
// store is the source of truth; whatever the policy believed before is
// replaced. Call it once at startup, before serving, and then periodically
// (see SyncLoop) when other replicas share the store.
//
// A reward handled here is applied to the policy right after it is written to
// the store, so a Sync that runs in between can count that reward twice. The
// next Sync replaces the state again and corrects it, so the error never
// lasts longer than one interval.
func (s *Server) Sync(ctx context.Context) error {
	totals, err := s.store.Totals(ctx)
	if err != nil {
		return fmt.Errorf("load totals: %w", err)
	}
	pending, err := s.store.PendingCounts(ctx)
	if err != nil {
		return fmt.Errorf("load pending counts: %w", err)
	}
	if err := s.policy.Restore(totals); err != nil {
		return fmt.Errorf("restore policy: %w", err)
	}
	return s.policy.SetPending(pending)
}

// SyncLoop calls Sync every interval until ctx is cancelled. Failures are
// logged and retried at the next tick: a replica that cannot reach the store
// keeps serving with the state it has, which is stale but usable.
func (s *Server) SyncLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if err := s.Sync(ctx); err != nil && ctx.Err() == nil {
			s.log.Warn("sync failed, continuing with stale state", "err", err)
		}
	}
}

// ExpireLoop removes expired pending selections every interval until ctx is
// cancelled. If implicitReward is non-nil, each unrewarded expired selection
// is counted as a reward of that value, in the store and in the policy.
func (s *Server) ExpireLoop(ctx context.Context, interval time.Duration, implicitReward *float64) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		expired, err := s.store.Expire(ctx, implicitReward)
		if err != nil {
			if ctx.Err() == nil {
				s.log.Error("expire pending selections", "err", err)
			}
			continue
		}
		total := 0
		for arm, n := range expired {
			total += n
			for range n {
				if implicitReward != nil {
					s.policy.Update(arm, *implicitReward) // learns, and releases the pending slot
				} else {
					s.policy.Abandon(arm) // just releases the pending slot
				}
			}
		}
		if total > 0 {
			if s.metrics != nil {
				s.metrics.Expired(total)
			}
			// Log the value, not the pointer: slog would print an address.
			counted := any("ignored")
			if implicitReward != nil {
				counted = *implicitReward
			}
			s.log.Info("selections expired without a reward", "count", total, "counted_as", counted)
		}
	}
}
