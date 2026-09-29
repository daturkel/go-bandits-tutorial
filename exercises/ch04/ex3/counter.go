package ex3

// Counter counts how often each key has been seen. Its zero value is ready to
// use: `var c Counter[string]; c.Inc("a")` must work without a constructor.
type Counter[K comparable] struct {
	counts map[K]int
}

// Inc adds one to key's count.
func (c *Counter[K]) Inc(key K) {
	// TODO
}

// Get returns key's count, or 0 if it was never seen.
func (c *Counter[K]) Get(key K) int {
	// TODO
	return 0
}

// MostCommon returns the key with the highest count and that count. The bool
// is false when nothing has been counted. (Ties may return either key.)
func (c *Counter[K]) MostCommon() (K, int, bool) {
	// TODO
	var zero K
	return zero, 0, false
}
