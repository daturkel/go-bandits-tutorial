package extras

// Generate returns a channel that yields nums in order and is then closed.
// The values must be sent from a separate goroutine, so Generate returns
// immediately.
func Generate(nums ...int) <-chan int {
	// TASK E3
	return nil
}

// Square returns a channel that yields the square of every value received
// from in, in order, and is closed after in is closed.
func Square(in <-chan int) <-chan int {
	// TASK E3
	return nil
}
