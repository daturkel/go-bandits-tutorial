package extras

import "context"

// Fetch runs work in a new goroutine and returns its result, or ctx.Err() if
// ctx ends first.
//
// The starter leaks: when ctx ends first, the goroutine finishes its work and
// then blocks forever trying to send a result nobody will receive. Fix it
// without changing the signature, and without making Fetch wait for work.
func Fetch(ctx context.Context, work func() string) (string, error) {
	ch := make(chan string, 1) // room for the one result, so the send never blocks
	go func() {
		ch <- work()
	}()
	select {
	case v := <-ch:
		return v, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
