package bandit

import (
	"errors"
	"fmt"
)

// ArmTotals is the durable summary of one arm: how many rewards have been
// recorded for it and their sum. Every policy in this package can be rebuilt
// from these two numbers per arm, which makes them what a database stores
// and, later, what replicas exchange: totals from different places simply add.
type ArmTotals struct {
	Arm       int
	Pulls     int64
	RewardSum float64
}

// Restorer is implemented by policies that can be rebuilt from ArmTotals.
type Restorer interface {
	// Restore replaces the policy's learned state. totals must contain one
	// entry per arm, in order.
	Restore(totals []ArmTotals) error
}

// ErrBadTotals is wrapped by Restore when the totals cannot describe a valid
// state for this policy.
var ErrBadTotals = errors.New("invalid arm totals")

// checkTotals validates totals for a policy with nArms arms.
func checkTotals(totals []ArmTotals, nArms int) error {
	if len(totals) != nArms {
		return fmt.Errorf("%w: got %d arms, policy has %d", ErrBadTotals, len(totals), nArms)
	}
	for i, t := range totals {
		switch {
		case t.Arm != i:
			return fmt.Errorf("%w: entry %d is for arm %d", ErrBadTotals, i, t.Arm)
		case t.Pulls < 0:
			return fmt.Errorf("%w: arm %d has %d pulls", ErrBadTotals, i, t.Pulls)
		case !(t.RewardSum >= 0 && t.RewardSum <= float64(t.Pulls)+1e-9):
			return fmt.Errorf("%w: arm %d has reward sum %v for %d pulls", ErrBadTotals, i, t.RewardSum, t.Pulls)
		}
	}
	return nil
}

// Restore sets the counts and means from totals.
func (s *armStats) Restore(totals []ArmTotals) error {
	if err := checkTotals(totals, len(s.counts)); err != nil {
		return err
	}
	s.total = 0
	for i, t := range totals {
		s.counts[i] = int(t.Pulls)
		s.total += int(t.Pulls)
		s.means[i] = 0
		if t.Pulls > 0 {
			s.means[i] = t.RewardSum / float64(t.Pulls)
		}
	}
	return nil
}

// Restore sets each Beta posterior to 1 + successes, 1 + failures, treating a
// reward r as r successes and 1-r failures, exactly as Update does.
func (p *Thompson) Restore(totals []ArmTotals) error {
	if err := checkTotals(totals, len(p.alpha)); err != nil {
		return err
	}
	for i, t := range totals {
		p.alpha[i] = 1 + t.RewardSum
		p.beta[i] = 1 + float64(t.Pulls) - t.RewardSum
	}
	return nil
}

// Restore replaces the wrapped policy's state, holding the lock.
func (l *Locked) Restore(totals []ArmTotals) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.p.Restore(totals)
}
