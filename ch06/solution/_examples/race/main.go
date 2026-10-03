// Run with the race detector:
//
//	go run -race ./_examples/race           one policy shared by four goroutines
//	go run -race ./_examples/race -locked   the same, through bandit.Locked
package main

import (
	"flag"
	"fmt"
	"sync"

	"banditlab/bandit"
)

func main() {
	locked := flag.Bool("locked", false, "share the policy through bandit.Locked")
	flag.Parse()

	pol, err := bandit.NewEpsilonGreedy(3, 0.1, bandit.NewRNG(1, bandit.StreamPolicy))
	if err != nil {
		panic(err)
	}
	var shared bandit.Policy = pol
	if *locked {
		shared = bandit.NewLocked(pol)
	}

	// Four goroutines use the one policy at the same time.
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 1000 {
				shared.Update(shared.Select(), 1)
			}
		}()
	}
	wg.Wait()
	fmt.Println("done")
}
