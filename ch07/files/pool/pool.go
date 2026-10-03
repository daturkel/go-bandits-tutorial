// Package pool runs independent jobs on a fixed number of worker goroutines.
package pool

import (
	"context"
	"errors"
)

// Options tunes Run.
type Options struct {
	// Workers is the number of goroutines that run jobs at once. Zero or
	// negative means runtime.GOMAXPROCS(0). It never exceeds the job count.
	Workers int

	// OnDone, if set, is called after each successful job with the number of
	// jobs finished so far and the total. It is called from the goroutine that
	// called Run, one call at a time, so it needs no locking of its own.
	OnDone func(done, total int)
}

// Run calls f for every job using a bounded pool of goroutines and returns the
// results in job order, whatever order they finish in.
//
// The first job to fail with an ordinary error cancels the context passed to
// f, so jobs still running can stop early, and jobs not yet started are not
// started. Run returns that error; if several jobs failed, it returns the one
// with the lowest index. If the caller's ctx ends first, Run returns
// ctx.Err(). Either way, no goroutine keeps running after Run returns: the
// workers have stopped and the helper goroutines are only returning.
//
// f should watch ctx and return promptly when it is done.
func Run[J, R any](ctx context.Context, opts Options, jobs []J, f func(ctx context.Context, job J) (R, error)) ([]R, error) {
	// TASK 2: workers, a feeder and a collector; results in job order.
	// TASK 3: failures and cancellation.
	return nil, errors.New("pool.Run is not written yet")
}

// isContextErr reports whether err is, or wraps, a cancellation or a deadline.
func isContextErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
