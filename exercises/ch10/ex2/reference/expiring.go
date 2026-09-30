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
	now     func() time.Time
	entries map[K]entry[V]
}

type entry[V any] struct {
	value     V
	expiresAt time.Time
}

// NewExpiring returns an empty Expiring using the given clock.
func NewExpiring[K comparable, V any](now func() time.Time) *Expiring[K, V] {
	return &Expiring[K, V]{now: now, entries: make(map[K]entry[V])}
}

// Set stores v under k for ttl.
func (e *Expiring[K, V]) Set(k K, v V, ttl time.Duration) {
	e.entries[k] = entry[V]{value: v, expiresAt: e.now().Add(ttl)}
}

// Get returns the value for k if it has not expired.
func (e *Expiring[K, V]) Get(k K) (V, bool) {
	en, ok := e.entries[k]
	if !ok || !e.now().Before(en.expiresAt) {
		var zero V
		return zero, false
	}
	return en.value, true
}

// Sweep removes expired entries and reports how many there were.
func (e *Expiring[K, V]) Sweep() int {
	now := e.now()
	n := 0
	for k, en := range e.entries {
		if !now.Before(en.expiresAt) {
			delete(e.entries, k) // deleting during range is allowed in Go
			n++
		}
	}
	return n
}

// Len returns the number of stored entries, expired or not.
func (e *Expiring[K, V]) Len() int { return len(e.entries) }
