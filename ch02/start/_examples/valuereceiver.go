package main

import "banditlab/bandit"

func main() {
	// The methods have pointer receivers, so only *EpsilonGreedy satisfies Policy.
	var p bandit.Policy = bandit.EpsilonGreedy{}
	_ = p
}
