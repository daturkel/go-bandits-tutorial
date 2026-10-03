package extras

import "sync"

// ParallelMap returns [f(xs[0]), f(xs[1]), ...], calling f for every element
// in its own goroutine. The result must be in the same order as xs.
//
// Each goroutine should write only to its own slot of the result slice; then
// no locking is needed. Wait for all of them with a sync.WaitGroup.
func ParallelMap[T, U any](xs []T, f func(T) U) []U {
	out := make([]U, len(xs))
	var wg sync.WaitGroup
	for i, x := range xs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = f(x) // distinct index per goroutine: no data race
		}()
	}
	wg.Wait()
	return out
}
