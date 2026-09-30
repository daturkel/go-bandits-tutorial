package ex3

import (
	"context"
	"time"
)

// permanent wraps an error that retrying cannot fix.
type permanent struct{ err error }

func (p permanent) Error() string { return p.err.Error() }
func (p permanent) Unwrap() error { return p.err }

// Permanent marks err as not worth retrying (a bad password, say, as opposed
// to a database that is still starting up).
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanent{err}
}

// Retry calls f until it succeeds, at most attempts times.
//
//   - Between failed attempts it waits with exponential backoff: base, then
//     2*base, 4*base, and so on, each capped at max. It waits by calling
//     sleep(ctx, d), which returns ctx.Err() if the context ends first
//     (production code passes SleepContext; tests pass a fake).
//   - It stops at once, returning the error, if f returns an error marked with
//     Permanent, or if sleep returns an error (the context ended).
//   - After the last attempt fails it returns that attempt's error, without
//     sleeping again.
//   - f receives ctx and the attempt number, starting at 1.
func Retry(ctx context.Context, attempts int, base, max time.Duration,
	sleep func(ctx context.Context, d time.Duration) error,
	f func(ctx context.Context, attempt int) error) error {
	// TODO
	return nil
}

// SleepContext waits for d or until ctx ends, whichever is first. It returns
// nil after a full wait and ctx.Err() otherwise.
func SleepContext(ctx context.Context, d time.Duration) error {
	// TODO
	return nil
}
