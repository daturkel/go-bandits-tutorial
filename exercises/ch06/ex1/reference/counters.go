package ex1

import (
	"sync"
	"sync/atomic"
)

// LockedCounter is a counter safe for use by many goroutines, using a mutex.
// The starter is not safe. Add a sync.Mutex field and use it.
type LockedCounter struct {
	mu sync.Mutex // guards n
	n  int
}

// Inc adds one.
func (c *LockedCounter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
}

// Value returns the current count.
func (c *LockedCounter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// AtomicCounter does the same without a mutex, using sync/atomic. Use one of
// the typed values such as atomic.Int64.
type AtomicCounter struct {
	n atomic.Int64
}

// Inc adds one.
func (c *AtomicCounter) Inc() {
	c.n.Add(1)
}

// Value returns the current count.
func (c *AtomicCounter) Value() int {
	return int(c.n.Load())
}
