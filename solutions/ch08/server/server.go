// Package server exposes a bandit policy over HTTP.
package server

import (
	"log/slog"
	"net/http"

	"banditlab/bandit"
)

// Server serves select and reward requests for one policy.
type Server struct {
	policy  *bandit.Locked
	pending *pendingStore
	log     *slog.Logger
}

// New returns a Server driving policy. The policy must not be used elsewhere
// afterwards: the server wraps it in a lock.
func New(policy bandit.SnapshotPolicy, log *slog.Logger) *Server {
	return &Server{
		policy:  bandit.NewLocked(policy),
		pending: newPendingStore(),
		log:     log,
	}
}

// Handler returns the routes wrapped in the standard middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /select", s.handleSelect)
	mux.HandleFunc("POST /reward", s.handleReward)
	mux.HandleFunc("GET /stats", s.handleStats)
	mux.HandleFunc("GET /healthz", handleHealth)

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

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
