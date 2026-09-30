package ex2

import "time"

// Expiring is a map whose entries disappear after a time-to-live: the core of
// an in-memory "pending selections" table.
//
//   - Set(k, v, ttl) stores v under k, valid for ttl from now. Setting an
//     existing key replaces its value and its deadline.
//   - Get(k) returns the value and true while it is valid. At or after the
//     deadline it returns the zero value and false.
//   - Sweep() deletes every expired entry and returns how many it deleted.
//   - Len() counts entries still stored, including expired ones that have not
//     been swept yet.
//
// It does not need to be safe for concurrent use. now is the clock.
type Expiring[K comparable, V any] struct {
	now func() time.Time
	// TODO: add the state you need.
}

// NewExpiring returns an empty Expiring using the given clock.
func NewExpiring[K comparable, V any](now func() time.Time) *Expiring[K, V] {
	return &Expiring[K, V]{now: now}
}

// Set stores v under k for ttl.
func (e *Expiring[K, V]) Set(k K, v V, ttl time.Duration) {
	// TODO
}

// Get returns the value for k if it has not expired.
func (e *Expiring[K, V]) Get(k K) (V, bool) {
	var zero V
	// TODO
	return zero, false
}

// Sweep removes expired entries and reports how many there were.
func (e *Expiring[K, V]) Sweep() int {
	// TODO
	return 0
}

// Len returns the number of stored entries, expired or not.
func (e *Expiring[K, V]) Len() int {
	// TODO
	return 0
}
