package ex1

// ParallelMap returns [f(xs[0]), f(xs[1]), ...], calling f for every element
// in its own goroutine. The result must be in the same order as xs.
//
// Each goroutine should write only to its own slot of the result slice; then
// no locking is needed. Wait for all of them with a sync.WaitGroup.
func ParallelMap[T, U any](xs []T, f func(T) U) []U {
	// TODO
	return nil
}
