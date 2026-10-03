// TASK E1: see the chapter page.

package extras

// LockedCounter is a counter safe for use by many goroutines, using a mutex.
// The starter is not safe. Add a sync.Mutex field and use it.
type LockedCounter struct {
	n int
}

// Inc adds one.
func (c *LockedCounter) Inc() {
	c.n++
}

// Value returns the current count.
func (c *LockedCounter) Value() int {
	return c.n
}

// AtomicCounter does the same without a mutex, using sync/atomic. Use one of
// the typed values such as atomic.Int64.
type AtomicCounter struct {
	n int
}

// Inc adds one.
func (c *AtomicCounter) Inc() {
	c.n++
}

// Value returns the current count.
func (c *AtomicCounter) Value() int {
	return c.n
}
