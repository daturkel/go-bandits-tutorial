package ex1

// Policy is the interface from the chapter, repeated here so the exercise
// stands alone.
type Policy interface {
	Name() string
	Select() int
	Update(arm int, reward float64)
}

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
