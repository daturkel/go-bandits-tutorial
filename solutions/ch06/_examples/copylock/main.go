package main

import (
	"fmt"

	"banditlab/bandit"
)

// report takes a Locked by value, which copies the mutex inside it.
func report(l bandit.Locked) {
	fmt.Println(l.Name())
}

func main() {
	pol, _ := bandit.NewUCB1(3)
	l := bandit.NewLocked(pol)
	report(*l)
}
