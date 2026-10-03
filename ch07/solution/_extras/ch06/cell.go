package extras

import "sync/atomic"

// Cell holds a value that many goroutines can read and replace without locks.
// Readers get an immutable snapshot: Load returns a copy, and writers never
// modify the stored value in place. They publish a new one.
type Cell[T any] struct {
	p atomic.Pointer[T]
}

// Load returns the current value, or the zero value if nothing was stored.
func (c *Cell[T]) Load() T {
	if v := c.p.Load(); v != nil {
		return *v
	}
	var zero T
	return zero
}

// Store replaces the value.
func (c *Cell[T]) Store(v T) {
	c.p.Store(&v)
}

// Update replaces the value with f(current) and returns the new value. It
// must be atomic: if another goroutine changes the cell between this call
// reading and writing, the change must not be lost. Use a compare-and-swap
// loop: load the pointer, compute the new value, and CompareAndSwap it in;
// if the swap fails, start again. f may therefore run more than once and must
// not have side effects.
//
// The starter reads and then writes as two separate steps, which loses updates.
func (c *Cell[T]) Update(f func(T) T) T {
	for {
		old := c.p.Load()
		var cur T
		if old != nil {
			cur = *old
		}
		next := f(cur)
		if c.p.CompareAndSwap(old, &next) {
			return next
		}
	}
}
