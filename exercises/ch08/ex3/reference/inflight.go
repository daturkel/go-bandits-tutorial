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
	sem := make(chan struct{}, n)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case sem <- struct{}{}: // got a slot
				defer func() { <-sem }()
				next.ServeHTTP(w, r)
			default: // all slots busy: refuse instead of queueing
				w.Header().Set("Retry-After", "1")
				http.Error(w, "server busy", http.StatusServiceUnavailable)
			}
		})
	}
}
