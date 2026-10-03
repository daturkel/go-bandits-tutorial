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
	// TASK 1: one ArmStat per arm, in arm order, built from s.counts and s.means.
	return nil
}

// Snapshot derives pulls and mean from the Beta parameters (each started at 1).
func (p *Thompson) Snapshot() []ArmStat {
	// TASK 1: alpha and beta each started at 1, so the pulls of an arm are
	// alpha + beta - 2, and the posterior mean is alpha / (alpha + beta).
	return nil
}
