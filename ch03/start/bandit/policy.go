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

// PolicyNames lists the policy names NewPolicy understands.
func PolicyNames() []string { return []string{"epsgreedy", "ucb1"} }

const defaultEpsilon = 0.1

// NewPolicy builds a policy from a spec: a name with an optional argument,
// such as "ucb1", "epsgreedy" or "epsgreedy:0.05".
func NewPolicy(spec string, nArms int, rng *rand.Rand) (Policy, error) {
	// TASK 5: this version only understands the plain names, and it has a bug
	// waiting to happen. Make it handle specs properly:
	//  - split spec at the first ":" (strings.Cut) into a name and an argument;
	//  - "epsgreedy" uses defaultEpsilon, or the argument parsed with
	//    strconv.ParseFloat; a bad number is an error that names the spec and
	//    wraps the strconv error;
	//  - "ucb1" takes no argument: giving one is an error;
	//  - an unknown name is an error wrapping ErrUnknownPolicy, and its message
	//    lists PolicyNames();
	//  - if a constructor fails, return a plain nil Policy and its error.
	//    Never return the constructor's nil pointer as the Policy.
	switch spec {
	case "epsgreedy":
		p, _ := NewEpsilonGreedy(nArms, defaultEpsilon, rng)
		return p, nil
	case "ucb1":
		p, _ := NewUCB1(nArms)
		return p, nil
	default:
		return nil, fmt.Errorf("unknown policy %q", spec)
	}
}
