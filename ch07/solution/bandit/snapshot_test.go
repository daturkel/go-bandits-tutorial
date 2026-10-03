package bandit

import (
	"math"
	"testing"
)

// Every policy reports one ArmStat per arm, in arm order, with the number of
// rewards it has seen for that arm. Mean is the policy's own estimate; after
// one success and one failure on an arm, every policy here estimates 0.5.
func TestSnapshotCountsPullsAndMeans(t *testing.T) {
	for _, spec := range PolicyNames() {
		t.Run(spec, func(t *testing.T) {
			p := mustSnapshot(t, spec, 1)
			p.Update(1, 1)
			p.Update(1, 0)
			p.Update(2, 1)
			got := p.Snapshot()
			if len(got) != 4 {
				t.Fatalf("Snapshot has %d entries, want 4", len(got))
			}
			for arm, want := range []int{0, 2, 1, 0} {
				if got[arm].Arm != arm || got[arm].Pulls != want {
					t.Errorf("entry %d = %+v, want arm %d with %d pulls", arm, got[arm], arm, want)
				}
			}
			if math.Abs(got[1].Mean-0.5) > 1e-9 {
				t.Errorf("arm 1 mean = %v, want 0.5", got[1].Mean)
			}
		})
	}
}

// The caller owns what Snapshot returns: changing it must not change the policy.
func TestSnapshotReturnsACopy(t *testing.T) {
	for _, spec := range PolicyNames() {
		p := mustSnapshot(t, spec, 1)
		p.Update(0, 1)
		snap := p.Snapshot()
		snap[0].Pulls = 99
		if again := p.Snapshot(); again[0].Pulls != 1 {
			t.Errorf("%s: changing a snapshot changed the policy (pulls now %d)", spec, again[0].Pulls)
		}
	}
}
