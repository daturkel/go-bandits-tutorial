package extras

// Generate returns a channel that yields nums in order and is then closed.
// The values must be sent from a separate goroutine, so Generate returns
// immediately.
func Generate(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			out <- n
		}
	}()
	return out
}

// Square returns a channel that yields the square of every value received
// from in, in order, and is closed after in is closed.
func Square(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			out <- n * n
		}
	}()
	return out
}
