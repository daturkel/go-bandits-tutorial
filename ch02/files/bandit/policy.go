package bandit

import (
	"fmt"
	"math/rand/v2"
)

// Policy decides which arm to pull next and learns from the outcome.
//
// Nothing here says which types implement it: any type with the right
// methods qualifies.
type Policy interface {
	// TASK 1: list the three methods every policy has: Select, Update and Name.
	// Their signatures are on the chapter page.
}

// PolicyNames lists the policies NewPolicy understands.
func PolicyNames() []string { return []string{"epsgreedy", "ucb1"} }

// NewPolicy builds a policy by name for an environment with nArms arms.
// epsilon and rng are used by the policies that need them.
func NewPolicy(name string, nArms int, epsilon float64, rng *rand.Rand) (Policy, error) {
	// TASK 5: switch on name. "epsgreedy" and "ucb1" build the matching
	// policy and return it with a nil error. Any other name returns a nil
	// Policy and an error that mentions the name.
	return nil, fmt.Errorf("NewPolicy is not written yet (task 5): %q", name)
}
