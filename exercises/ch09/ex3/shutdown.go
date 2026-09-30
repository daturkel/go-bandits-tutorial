package ex3

import (
	"net/http"
	"time"
)

// Shutdown stops srv gracefully. It stops accepting new connections and
// waits up to grace for requests already in progress to finish.
//
//   - If everything finishes in time, return nil.
//   - If the grace period runs out first, close the remaining connections
//     immediately and return an error that wraps context.DeadlineExceeded
//     (errors.Is(err, context.DeadlineExceeded) must be true).
//
// (*http.Server).Shutdown takes a context and does the waiting; Close drops
// connections at once.
func Shutdown(srv *http.Server, grace time.Duration) error {
	// TODO
	return nil
}
