package extras

import (
	"context"
	"errors"
)

// First runs every fn concurrently and returns the value of the first one to
// succeed (return a nil error).
//
//   - As soon as one succeeds, cancel the context passed to the others.
//   - Do not return until every fn has returned, so nothing is left running.
//   - Failures from functions that lose the race are ignored.
//   - If none succeeds, return the zero value and all the errors joined
//     (errors.Join). With no functions at all, return an error.
//
// Hint: give each fn a goroutine and send results on a channel with room for
// every result, so a goroutine never blocks on a send.
func First[T any](ctx context.Context, fns ...func(context.Context) (T, error)) (T, error) {
	var zero T
	if len(fns) == 0 {
		return zero, errors.New("First: no functions to run")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		v   T
		err error
	}
	results := make(chan result, len(fns)) // room for every result: no sender ever blocks
	for _, fn := range fns {
		go func() {
			v, err := fn(ctx)
			results <- result{v, err}
		}()
	}

	var (
		winner *T
		errs   []error
	)
	for range fns { // collect all of them, so no goroutine outlives the call
		r := <-results
		switch {
		case r.err == nil && winner == nil:
			winner = &r.v
			cancel() // tell the others to stop
		case r.err != nil:
			errs = append(errs, r.err)
		}
	}
	if winner != nil {
		return *winner, nil
	}
	return zero, errors.Join(errs...)
}
