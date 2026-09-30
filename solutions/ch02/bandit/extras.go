package bandit

import "fmt"

// RoundRobin pulls arm 0, 1, ..., n-1, 0, 1, ... and ignores rewards. It is a
// useful baseline: any learning policy should beat it.
type RoundRobin struct {
	nArms int
	next  int
}

// NewRoundRobin returns a RoundRobin for nArms arms.
func NewRoundRobin(nArms int) *RoundRobin {
	return &RoundRobin{nArms: nArms}
}

// Name returns "round-robin".
func (r *RoundRobin) Name() string { return "round-robin" }

// Select returns the next arm in the cycle.
func (r *RoundRobin) Select() int {
	arm := r.next
	r.next = (r.next + 1) % r.nArms
	return arm
}

// Update does nothing: round-robin does not learn.
func (r *RoundRobin) Update(arm int, reward float64) {}

// Logged wraps a Policy and records every call in Log, as "select 2" and
// "update 1 1" (arm, then reward printed with %g). Everything else is
// delegated to the wrapped policy unchanged.
type Logged struct {
	Policy // embedded: Name, Select and Update are promoted from it
	Log    []string
}

// NewLogged wraps p.
func NewLogged(p Policy) *Logged { return &Logged{Policy: p} }

// Select records the chosen arm.
func (l *Logged) Select() int {
	arm := l.Policy.Select()
	l.Log = append(l.Log, fmt.Sprintf("select %d", arm))
	return arm
}

// Update records the observation, then passes it on.
func (l *Logged) Update(arm int, reward float64) {
	l.Log = append(l.Log, fmt.Sprintf("update %d %g", arm, reward))
	l.Policy.Update(arm, reward)
}

// Tally counts pulls per arm and in total.
//
// Methods that change the struct need pointer receivers. With value
// receivers each call works on a copy, so the shared slice gets updated but
// the copied total is thrown away.
type Tally struct {
	counts []int
	total  int
}

// NewTally returns a Tally for nArms arms.
func NewTally(nArms int) *Tally {
	return &Tally{counts: make([]int, nArms)}
}

// Add records one pull of arm.
func (t *Tally) Add(arm int) {
	t.counts[arm]++
	t.total++
}

// Total returns the number of pulls recorded.
func (t *Tally) Total() int { return t.total }

// Count returns the number of pulls of arm.
func (t *Tally) Count(arm int) int { return t.counts[arm] }
