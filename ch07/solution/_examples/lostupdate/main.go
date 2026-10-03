package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	goroutines = 100
	perRoutine = 10_000
)

// run starts goroutines that each call inc perRoutine times.
func run(inc func()) {
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perRoutine {
				inc()
			}
		}()
	}
	wg.Wait()
}

func main() {
	want := goroutines * perRoutine

	plain := 0
	run(func() { plain++ })
	fmt.Printf("plain int:    %8d (want %d)\n", plain, want)

	var mu sync.Mutex
	locked := 0
	run(func() {
		mu.Lock()
		locked++
		mu.Unlock()
	})
	fmt.Printf("with mutex:   %8d\n", locked)

	var atom atomic.Int64
	run(func() { atom.Add(1) })
	fmt.Printf("with atomic:  %8d\n", atom.Load())
}
