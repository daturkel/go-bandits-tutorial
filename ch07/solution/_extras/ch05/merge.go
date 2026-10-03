package extras

import "sync"

// Merge fans several channels into one. Every value received from any input
// is sent on the returned channel, in whatever order values arrive. The
// returned channel is closed once all inputs have been closed and drained.
//
// With no inputs the returned channel is closed immediately.
func Merge(chans ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup
	for _, ch := range chans {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for v := range ch {
				out <- v
			}
		}()
	}
	go func() {
		wg.Wait()
		close(out) // only after every forwarding goroutine has finished
	}()
	return out
}
