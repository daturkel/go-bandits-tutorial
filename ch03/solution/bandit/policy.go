package bandit

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
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

// PolicyNames lists the policy names NewPolicy understands.
func PolicyNames() []string { return []string{"epsgreedy", "ucb1"} }

const defaultEpsilon = 0.1

// NewPolicy builds a policy from a spec: a name with an optional argument,
// such as "ucb1", "epsgreedy" or "epsgreedy:0.05".
func NewPolicy(spec string, nArms int, rng *rand.Rand) (Policy, error) {
	name, arg, hasArg := strings.Cut(spec, ":")
	switch name {
	case "epsgreedy":
		epsilon := defaultEpsilon
		if hasArg {
			var err error
			if epsilon, err = strconv.ParseFloat(arg, 64); err != nil {
				return nil, fmt.Errorf("policy %q: bad epsilon: %w", spec, err)
			}
		}
		p, err := NewEpsilonGreedy(nArms, epsilon, rng)
		if err != nil {
			return nil, err // not `return p, err`: p is a nil *EpsilonGreedy, and a Policy holding it would not be nil
		}
		return p, nil
	case "ucb1":
		if hasArg {
			return nil, fmt.Errorf("policy %q: ucb1 takes no argument", spec)
		}
		p, err := NewUCB1(nArms)
		if err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, fmt.Errorf("%w %q (known: %s)", ErrUnknownPolicy, name, strings.Join(PolicyNames(), ", "))
	}
}
