package ex1

import "time"

// Backoff returns how long to wait before retry number attempt (0 for the
// first retry): base, then 2*base, 4*base, and so on, never more than max.
// A service that reconnects to a dead peer with a fixed short delay hammers it
// the moment it comes back; growing delays give it room to recover.
//
// It must not overflow: attempt can be large (a peer that has been down for
// hours). If base <= 0 or max <= 0 it returns 0; if base > max it returns max.
func Backoff(attempt int, base, max time.Duration) time.Duration {
	if base <= 0 || max <= 0 {
		return 0
	}
	d := base
	for range attempt { // a negative attempt runs zero times
		if d > max/2 { // doubling would exceed max; checked this way so it cannot overflow
			return max
		}
		d *= 2
	}
	return min(d, max)
}
