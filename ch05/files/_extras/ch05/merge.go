package extras

// Merge fans several channels into one. Every value received from any input
// is sent on the returned channel, in whatever order values arrive. The
// returned channel is closed once all inputs have been closed and drained.
//
// With no inputs the returned channel is closed immediately.
func Merge(chans ...<-chan int) <-chan int {
	// TASK E2
	return nil
}
