package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

var errTimeout = errors.New("timed out")

// fetchLeaky gives up after timeout, but the worker goroutine it started is
// left blocked on a send that nobody will ever receive.
func fetchLeaky(timeout time.Duration) (string, error) {
	ch := make(chan string) // unbuffered
	go func() {
		time.Sleep(20 * time.Millisecond) // slow work
		ch <- "result"                    // blocks forever if the caller left
	}()
	select {
	case v := <-ch:
		return v, nil
	case <-time.After(timeout):
		return "", errTimeout
	}
}

// fetchFixed is identical except that the channel has room for the result,
// so the worker can always finish its send and exit.
func fetchFixed(timeout time.Duration) (string, error) {
	ch := make(chan string, 1) // buffered
	go func() {
		time.Sleep(20 * time.Millisecond)
		ch <- "result"
	}()
	select {
	case v := <-ch:
		return v, nil
	case <-time.After(timeout):
		return "", errTimeout
	}
}

func main() {
	for range 1000 {
		fetchFixed(time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	fmt.Println("after 1000 timed-out fixed calls:", runtime.NumGoroutine(), "goroutine(s)")

	for range 1000 {
		fetchLeaky(time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	fmt.Println("after 1000 timed-out leaky calls:", runtime.NumGoroutine(), "goroutine(s)")

	fmt.Println()
	fmt.Println("what is running:")
	pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
}
