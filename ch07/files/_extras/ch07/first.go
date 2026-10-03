package extras

import "context"

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
	// TASK E1
	var zero T
	return zero, nil
}
