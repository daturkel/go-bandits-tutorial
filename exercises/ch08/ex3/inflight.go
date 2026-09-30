package ex3

import "net/http"

// MaxInFlight returns middleware that allows at most n requests to be inside
// the wrapped handler at once. A request that arrives while n are already
// running must not wait: respond immediately with 503 Service Unavailable and
// the header `Retry-After: 1`, without calling next.
//
// Rejecting fast keeps a slow dependency from piling up unbounded goroutines
// and memory. Hint: a buffered channel is a counting semaphore, and a select
// with a default case is a non-blocking attempt to acquire it.
func MaxInFlight(n int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// TODO
		return next
	}
}
