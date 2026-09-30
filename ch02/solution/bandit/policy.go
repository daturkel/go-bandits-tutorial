package bandit

import (
	"fmt"
	"math/rand/v2"
)

// Policy decides which arm to pull next and learns from the outcome.
//
// Nothing here says which types implement it: any type with these three
// methods qualifies.
type Policy interface {
	Name() string
	Select() int
	Update(arm int, reward float64)
}

// PolicyNames lists the policies NewPolicy understands.
func PolicyNames() []string { return []string{"epsgreedy", "ucb1"} }

// NewPolicy builds a policy by name for an environment with nArms arms.
func NewPolicy(name string, nArms int, epsilon float64, rng *rand.Rand) (Policy, error) {
	switch name {
	case "epsgreedy":
		return NewEpsilonGreedy(nArms, epsilon, rng), nil
	case "ucb1":
		return NewUCB1(nArms), nil
	default:
		return nil, fmt.Errorf("unknown policy %q", name)
	}
}
