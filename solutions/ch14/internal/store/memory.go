package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"banditlab/internal/bandit"
)

// Memory is a Store that lives in the process. It is fast, needs nothing
// installed, and loses everything on exit, which suits tests and local runs.
type Memory struct {
	now func() time.Time // replaceable, so tests can move time

	mu      sync.Mutex
	pending map[string]*entry // guarded by mu
	totals  []bandit.ArmTotals
}

type entry struct {
	arm       int
	expiresAt time.Time
	rewarded  bool
}

// MemoryOption adjusts a Memory store.
type MemoryOption func(*Memory)

// WithClock replaces time.Now.
func WithClock(now func() time.Time) MemoryOption {
	return func(m *Memory) { m.now = now }
}

// NewMemory returns an empty store for arms arms.
func NewMemory(arms int, opts ...MemoryOption) *Memory {
	m := &Memory{
		now:     time.Now,
		pending: make(map[string]*entry),
		totals:  make([]bandit.ArmTotals, arms),
	}
	for i := range m.totals {
		m.totals[i].Arm = i
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *Memory) AddPending(_ context.Context, id string, arm int, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if arm < 0 || arm >= len(m.totals) {
		return fmt.Errorf("arm %d out of range", arm)
	}
	if _, dup := m.pending[id]; dup {
		return fmt.Errorf("request id %q already exists", id)
	}
	m.pending[id] = &entry{arm: arm, expiresAt: m.now().Add(ttl)}
	return nil
}

func (m *Memory) Reward(_ context.Context, id string, reward float64) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.pending[id]
	switch {
	case !ok || !m.now().Before(e.expiresAt):
		return 0, ErrUnknownID
	case e.rewarded:
		return 0, ErrAlreadyRewarded
	}
	e.rewarded = true
	m.totals[e.arm].Pulls++
	m.totals[e.arm].RewardSum += reward
	return e.arm, nil
}

func (m *Memory) Expire(_ context.Context, implicitReward *float64) (map[int]int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	expired := map[int]int{}
	now := m.now()
	for id, e := range m.pending {
		if now.Before(e.expiresAt) {
			continue
		}
		delete(m.pending, id)
		if e.rewarded {
			continue
		}
		expired[e.arm]++
		if implicitReward != nil {
			m.totals[e.arm].Pulls++
			m.totals[e.arm].RewardSum += *implicitReward
		}
	}
	return expired, nil
}

func (m *Memory) Totals(context.Context) ([]bandit.ArmTotals, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]bandit.ArmTotals, len(m.totals))
	copy(out, m.totals)
	return out, nil
}

func (m *Memory) PendingCounts(context.Context) ([]int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	counts := make([]int, len(m.totals))
	now := m.now()
	for _, e := range m.pending {
		if !e.rewarded && now.Before(e.expiresAt) {
			counts[e.arm]++
		}
	}
	return counts, nil
}

func (m *Memory) Ping(context.Context) error { return nil }
func (m *Memory) Close() error               { return nil }

var _ Store = (*Memory)(nil)
