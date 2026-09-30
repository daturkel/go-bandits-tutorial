package bandit

import (
	"sync"
	"sync/atomic"
)

// Locked makes a policy safe to share between goroutines by allowing one
// call into it at a time. A Locked must not be copied after first use; pass
// and store it as a *Locked.
type Locked struct {
	mu sync.Mutex // guards p
	p  SnapshotPolicy

	// Call counters live outside the mutex: they are atomics, so they can be
	// read at any moment without waiting for a Select that holds the lock.
	selects atomic.Int64
	updates atomic.Int64
}

// NewLocked wraps p. After this call, use p only through the returned value.
func NewLocked(p SnapshotPolicy) *Locked {
	return &Locked{p: p}
}

// Name reports the wrapped policy's name. It is fixed for the policy's
// lifetime, so it needs no lock.
func (l *Locked) Name() string { return l.p.Name() }

// Select chooses an arm.
func (l *Locked) Select() int {
	l.selects.Add(1)
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.p.Select()
}

// Update records a reward.
func (l *Locked) Update(arm int, reward float64) {
	l.updates.Add(1)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.p.Update(arm, reward)
}

// Abandon releases a pending selection that will never be rewarded. Policies
// that do not track pending selections ignore it.
func (l *Locked) Abandon(arm int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if pt, ok := l.p.(PendingTracker); ok {
		pt.Abandon(arm)
	}
}

// SetPending replaces the pending counts of policies that track them, and does
// nothing for those that do not.
func (l *Locked) SetPending(counts []int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if pt, ok := l.p.(PendingTracker); ok {
		return pt.SetPending(counts)
	}
	return nil
}

// Snapshot returns a consistent copy of the policy's per-arm state.
func (l *Locked) Snapshot() []ArmStat {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.p.Snapshot()
}

// Calls reports how many Select and Update calls have been made. It never
// blocks.
func (l *Locked) Calls() (selects, updates int64) {
	return l.selects.Load(), l.updates.Load()
}
