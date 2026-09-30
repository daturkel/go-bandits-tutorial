package main

import (
	"fmt"
	"sync"

	"banditlab/bandit"
)

func main() {
	// One policy shared by four goroutines, with no synchronisation.
	pol, err := bandit.NewEpsilonGreedy(3, 0.1, bandit.NewRNG(1, bandit.StreamPolicy))
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 1000 {
				pol.Update(pol.Select(), 1)
			}
		}()
	}
	wg.Wait()
	fmt.Println("done")
}
