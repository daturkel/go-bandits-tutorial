package bandit

import (
	"fmt"
	"math"
)

// UCB1 pulls the arm with the highest optimistic estimate: its mean plus a
// bonus that shrinks as the arm is pulled more often. It uses no randomness.
//
// Selections whose reward has not arrived yet count towards an arm's pull
// count in the bonus (but not in its mean, since their outcome is unknown).
// Without that, a burst of requests before any reward comes back would all
// see the same scores and all get the same arm. Every Select must therefore be
// followed, eventually, by Update (the reward arrived) or Abandon (it never
// will).
type UCB1 struct {
	armStats
	scores        []float64 // scratch space reused by every Select
	pending       []int     // selections per arm still waiting for a reward
	pendingTotal  int
	ignorePending bool // the textbook algorithm: see IgnorePending
}

// UCB1Option adjusts a UCB1 policy.
type UCB1Option func(*UCB1)

// IgnorePending gives the textbook UCB1, which only counts selections whose
// reward has arrived. It is here so the cost of ignoring pending selections
// can be measured; do not use it behind a service with delayed feedback.
func IgnorePending() UCB1Option {
	return func(p *UCB1) { p.ignorePending = true }
}

// NewUCB1 returns a UCB1 policy for nArms arms.
func NewUCB1(nArms int, opts ...UCB1Option) (*UCB1, error) {
	if nArms < 1 {
		return nil, ErrNoArms
	}
	p := &UCB1{
		armStats: newArmStats(nArms),
		scores:   make([]float64, nArms),
		pending:  make([]int, nArms),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p, nil
}

// Name identifies the policy in reports.
func (p *UCB1) Name() string {
	if p.ignorePending {
		return "ucb1-naive"
	}
	return "ucb1"
}

// Select tries every arm once, then maximises mean + sqrt(2 ln t / n), where
// n and t include selections still waiting for their reward.
func (p *UCB1) Select() int {
	arm := p.choose()
	if !p.ignorePending {
		p.pending[arm]++
		p.pendingTotal++
	}
	return arm
}

func (p *UCB1) choose() int {
	for arm, n := range p.counts {
		if n+p.pending[arm] == 0 {
			return arm
		}
	}
	t := p.total + p.pendingTotal
	for arm, n := range p.counts {
		n += p.pending[arm]
		p.scores[arm] = p.means[arm] + math.Sqrt(2*math.Log(float64(t))/float64(n))
	}
	return Argmax(p.scores, nil)
}

// Update learns from a reward and releases the arm's pending selection. A
// reward with no matching pending selection (one that another replica made)
// is still learned from.
func (p *UCB1) Update(arm int, reward float64) {
	p.release(arm)
	p.armStats.Update(arm, reward)
}

// Abandon releases a pending selection that will never receive a reward.
func (p *UCB1) Abandon(arm int) { p.release(arm) }

func (p *UCB1) release(arm int) {
	if p.pending[arm] > 0 {
		p.pending[arm]--
		p.pendingTotal--
	}
}

// SetPending replaces the pending counts, for example with the number of
// selections all replicas together are waiting on.
func (p *UCB1) SetPending(counts []int) error {
	if len(counts) != len(p.pending) {
		return fmt.Errorf("%w: got %d pending counts, policy has %d arms", ErrBadTotals, len(counts), len(p.pending))
	}
	total := 0
	for _, n := range counts {
		if n < 0 {
			return fmt.Errorf("%w: negative pending count %d", ErrBadTotals, n)
		}
		total += n
	}
	if p.ignorePending {
		return nil
	}
	copy(p.pending, counts)
	p.pendingTotal = total
	return nil
}

// Snapshot adds the pending counts to the shared bookkeeping's view.
func (p *UCB1) Snapshot() []ArmStat {
	out := p.armStats.Snapshot()
	for arm := range out {
		out[arm].Pending = p.pending[arm]
	}
	return out
}
