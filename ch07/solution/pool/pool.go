// Package pool runs independent jobs on a fixed number of worker goroutines.
package pool

import (
	"context"
	"errors"
	"runtime"
	"sync"
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
	parent := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	workers = min(workers, len(jobs))

	type outcome struct {
		index int
		res   R
		err   error
	}
	work := make(chan int)
	results := make(chan outcome)

	// Fan out: a fixed set of workers take job indexes from one channel.
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range work {
				res, err := f(ctx, jobs[i])
				results <- outcome{i, res, err}
			}
		}()
	}

	// Feed the workers, but stop as soon as the context is cancelled.
	go func() {
		defer close(work)
		for i := range jobs {
			select {
			case work <- i:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Fan in. Once every worker has exited, close results so the loop below ends.
	go func() {
		wg.Wait()
		close(results)
	}()

	out := make([]R, len(jobs))
	errs := make([]error, len(jobs))
	done := 0
	// This loop always drains results, so a worker is never stuck sending.
	for o := range results {
		out[o.index], errs[o.index] = o.res, o.err
		switch {
		case o.err == nil:
			done++
			if opts.OnDone != nil {
				opts.OnDone(done, len(jobs))
			}
		case !isContextErr(o.err):
			cancel() // an ordinary failure: stop everyone else
		}
	}

	for _, err := range errs {
		if err != nil && !isContextErr(err) {
			return nil, err
		}
	}
	if err := parent.Err(); err != nil {
		return nil, err
	}
	for _, err := range errs { // a job saw a context error the caller did not cause
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func isContextErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
