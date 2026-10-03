package main

import (
	"fmt"
	"sync"
)

func main() {
	// An unbuffered channel: a send waits for a receive, and vice versa.
	results := make(chan string)

	var wg sync.WaitGroup
	for _, name := range []string{"epsgreedy", "ucb1", "thompson"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- "finished " + name // blocks until main receives
		}()
	}
	// Close the channel when all senders are done, so the range below ends.
	go func() {
		wg.Wait()
		close(results)
	}()

	count := 0
	for range results { // the order messages arrive in varies from run to run
		count++
	}
	fmt.Println("received", count, "messages")

	// A buffered channel holds values without a waiting receiver.
	buf := make(chan int, 2)
	buf <- 1
	buf <- 2
	fmt.Println("buffered:", len(buf), "of", cap(buf))
	close(buf)
	for v := range buf { // a closed channel still yields what it holds
		fmt.Println("got", v)
	}
	v, ok := <-buf // then reports closed
	fmt.Println("after close:", v, ok)
}
