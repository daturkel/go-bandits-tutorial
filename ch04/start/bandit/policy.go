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
func PolicyNames() []string { return []string{"epsgreedy", "ucb1", "thompson"} }

const defaultEpsilon = 0.1

// NewPolicy builds a policy from a spec: a name with an optional argument,
// such as "ucb1", "thompson", "epsgreedy" or "epsgreedy:0.05".
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
		return asPolicy(NewEpsilonGreedy(nArms, epsilon, rng))
	case "ucb1":
		if hasArg {
			return nil, fmt.Errorf("policy %q: ucb1 takes no argument", spec)
		}
		return asPolicy(NewUCB1(nArms))
	case "thompson":
		if hasArg {
			return nil, fmt.Errorf("policy %q: thompson takes no argument", spec)
		}
		return asPolicy(NewThompson(nArms, rng))
	default:
		return nil, fmt.Errorf("%w %q (known: %s)", ErrUnknownPolicy, name, strings.Join(PolicyNames(), ", "))
	}
}

// asPolicy converts a constructor's (*T, error) result into (Policy, error).
// Returning the pair directly would wrap a nil *T in a non-nil Policy whenever
// the constructor fails; this returns a true nil instead. The parameter is the
// Policy interface itself: no type parameter is needed because all the helper
// does is check err and pass the value along.
func asPolicy(p Policy, err error) (Policy, error) {
	// TASK 3: if err is set, return a plain nil Policy and err. Otherwise
	// return p.
	return p, err
}
