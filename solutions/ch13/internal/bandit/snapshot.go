package bandit

// ArmStat summarises what a policy currently believes about one arm.
type ArmStat struct {
	Arm     int     `json:"arm"`
	Pulls   int     `json:"pulls"`
	Mean    float64 `json:"mean"`
	Pending int     `json:"pending"` // selections still waiting for a reward (UCB1 only)
}

// Snapshotter reports a policy's per-arm state. The returned slice is a copy:
// the caller may keep and modify it.
type Snapshotter interface {
	Snapshot() []ArmStat
}

// SnapshotPolicy is a Policy whose learned state can be read (Snapshot) and
// replaced (Restore).
type SnapshotPolicy interface {
	Policy
	Snapshotter
	Restorer
}

// Snapshot copies the counts and means. It is promoted to every policy that
// embeds armStats.
func (s *armStats) Snapshot() []ArmStat {
	out := make([]ArmStat, len(s.counts))
	for arm := range out {
		out[arm] = ArmStat{Arm: arm, Pulls: s.counts[arm], Mean: s.means[arm]}
	}
	return out
}

// PendingTracker is implemented by policies that account for selections that
// are still waiting for their reward.
type PendingTracker interface {
	// Abandon releases a pending selection that will never receive a reward.
	Abandon(arm int)
	// SetPending replaces the per-arm pending counts.
	SetPending(counts []int) error
}

// Snapshot derives pulls and mean from the Beta parameters (each started at 1).
func (p *Thompson) Snapshot() []ArmStat {
	out := make([]ArmStat, len(p.alpha))
	for arm := range out {
		out[arm] = ArmStat{
			Arm:   arm,
			Pulls: int(p.alpha[arm] + p.beta[arm] - 2),
			Mean:  p.alpha[arm] / (p.alpha[arm] + p.beta[arm]),
		}
	}
	return out
}
