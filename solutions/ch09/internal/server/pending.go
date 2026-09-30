package server

import (
	"errors"
	"sync"
)

var (
	errUnknownID       = errors.New("unknown request id")
	errAlreadyRewarded = errors.New("reward already recorded for this request id")
)

// pendingStore remembers which arm each selection returned, until its reward
// arrives. It lives in memory and only grows; chapter 10 adds expiry and
// persistence.
type pendingStore struct {
	mu      sync.Mutex
	entries map[string]*pending // guarded by mu
}

type pending struct {
	arm      int
	rewarded bool
}

func newPendingStore() *pendingStore {
	return &pendingStore{entries: make(map[string]*pending)}
}

// add records that the selection with this id returned arm.
func (s *pendingStore) add(id string, arm int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[id] = &pending{arm: arm}
}

// claim marks the id as rewarded and returns its arm. A second claim of the
// same id fails, so a retried request cannot count twice.
func (s *pendingStore) claim(id string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.entries[id]
	switch {
	case !ok:
		return 0, errUnknownID
	case p.rewarded:
		return 0, errAlreadyRewarded
	}
	p.rewarded = true
	return p.arm, nil
}
