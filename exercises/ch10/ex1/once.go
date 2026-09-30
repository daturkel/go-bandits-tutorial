package ex1

import (
	"sync"
	"time"
)

// Once runs an action at most once per key within a time window: the
// in-process version of "claim this reward id".
//
//   - Do(key, f) calls f only if no earlier call for key ran (or is running)
//     within the last ttl. It reports whether it called f.
//   - Of many concurrent Do calls with the same key, exactly one runs f; the
//     others return (false, nil) without waiting for it.
//   - If f returns an error, the claim is released: the error is returned
//     with executed == true, and a later Do for the same key may run f again.
//   - After ttl has passed since a successful run, the key is free again.
//
// now is the clock, injected so tests can move time.
type Once struct {
	now func() time.Time
	ttl time.Duration

	mu sync.Mutex
	// TODO: add the state you need, guarded by mu.
}

// NewOnce returns a Once with the given window and clock.
func NewOnce(ttl time.Duration, now func() time.Time) *Once {
	return &Once{now: now, ttl: ttl}
}

// Do runs f for key unless it has already been claimed.
func (o *Once) Do(key string, f func() error) (executed bool, err error) {
	// TODO
	return false, nil
}
