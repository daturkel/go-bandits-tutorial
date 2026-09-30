package bandit

// ArmStat summarises what a policy currently believes about one arm.
type ArmStat struct {
	Arm   int
	Pulls int
	Mean  float64
}

// Snapshotter reports a policy's per-arm state. The returned slice is a copy:
// the caller may keep and modify it.
type Snapshotter interface {
	Snapshot() []ArmStat
}

// SnapshotPolicy is a Policy that can also report its state.
type SnapshotPolicy interface {
	Policy
	Snapshotter
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
