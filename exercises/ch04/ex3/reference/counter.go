package ex3

// Counter counts how often each key has been seen. Its zero value is ready to
// use: `var c Counter[string]; c.Inc("a")` must work without a constructor.
type Counter[K comparable] struct {
	counts map[K]int
}

// Inc adds one to key's count.
func (c *Counter[K]) Inc(key K) {
	if c.counts == nil {
		c.counts = make(map[K]int) // created on first use, so the zero value works
	}
	c.counts[key]++
}

// Get returns key's count, or 0 if it was never seen.
func (c *Counter[K]) Get(key K) int {
	return c.counts[key] // reading a nil map is fine and yields 0
}

// MostCommon returns the key with the highest count and that count. The bool
// is false when nothing has been counted. (Ties may return either key.)
func (c *Counter[K]) MostCommon() (K, int, bool) {
	var best K
	bestN := 0
	for key, n := range c.counts {
		if n > bestN {
			best, bestN = key, n
		}
	}
	return best, bestN, bestN > 0
}
