// Package server exposes a bandit policy over HTTP.
package server

import (
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"banditlab/internal/bandit"
)

// Server serves select and reward requests for one policy.
type Server struct {
	policy      *bandit.Locked
	pending     *pendingStore
	log         *slog.Logger
	selectDelay time.Duration
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

// New returns a Server driving policy. The policy must not be used elsewhere
// afterwards: the server wraps it in a lock.
func New(policy bandit.SnapshotPolicy, log *slog.Logger, opts ...Option) *Server {
	s := &Server{
		policy:  bandit.NewLocked(policy),
		pending: newPendingStore(),
		log:     log,
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
	// logger, and the logger sees the 500 that recoverPanic writes.
	return Chain(mux, requestID, logging(s.log), recoverPanic(s.log))
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
	s.pending.add(id, arm)
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

	arm, err := s.pending.claim(req.RequestID)
	switch err {
	case nil:
	case errUnknownID:
		writeError(w, http.StatusNotFound, err.Error())
		return
	case errAlreadyRewarded:
		writeError(w, http.StatusConflict, err.Error())
		return
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	s.policy.Update(arm, *req.Reward)
	w.WriteHeader(http.StatusNoContent)
}

type statsResponse struct {
	Policy  string           `json:"policy"`
	Selects int64            `json:"selects"`
	Updates int64            `json:"updates"`
	Arms    []bandit.ArmStat `json:"arms"`
}

// handleStats reports what the policy currently believes.
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	selects, updates := s.policy.Calls()
	writeJSON(w, http.StatusOK, statsResponse{
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
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
