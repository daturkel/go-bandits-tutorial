package ex3

import "context"

// Latest holds the most recent value of something that changes, and lets
// goroutines wait for it to change. It is the pattern behind the
// aggregator's Watch stream, without gRPC.
//
// Every Set increases the version by one. The zero value is not usable; use
// NewLatest.
type Latest[T any] struct {
	// add whatever fields you need
}

// NewLatest returns a Latest holding initial at version 0.
func NewLatest[T any](initial T) *Latest[T] {
	// TODO
	return &Latest[T]{}
}

// Set replaces the value and wakes every goroutine blocked in Wait.
func (l *Latest[T]) Set(v T) {
	// TODO
}

// Wait returns the current value and its version as soon as the version is
// greater than after. If it already is, Wait returns immediately; otherwise
// it blocks until a Set, or until ctx ends, in which case it returns ctx's
// error. A caller that passes the version it last saw sees every change or a
// later state, and never blocks on a change it already has.
func (l *Latest[T]) Wait(ctx context.Context, after uint64) (T, uint64, error) {
	// TODO
	var zero T
	return zero, 0, nil
}
