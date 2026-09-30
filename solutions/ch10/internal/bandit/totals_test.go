package bandit

import (
	"errors"
	"math"
	"testing"
)

func sampleTotals() []ArmTotals {
	return []ArmTotals{
		{Arm: 0, Pulls: 0, RewardSum: 0},
		{Arm: 1, Pulls: 10, RewardSum: 7},
		{Arm: 2, Pulls: 4, RewardSum: 0.5}, // fractional rewards are allowed
		{Arm: 3, Pulls: 1, RewardSum: 1},
	}
}

func TestRestoreSetsState(t *testing.T) {
	for _, spec := range PolicyNames() {
		t.Run(spec, func(t *testing.T) {
			pol := mustSnapshot(t, spec, 1)
			if err := pol.Restore(sampleTotals()); err != nil {
				t.Fatal(err)
			}
			got := pol.Snapshot()
			// Thompson's mean is the posterior mean (1+sum)/(2+pulls); the
			// others report the plain average.
			wantMean := func(arm int) float64 {
				tot := sampleTotals()[arm]
				if spec == "thompson" {
					return (1 + tot.RewardSum) / (2 + float64(tot.Pulls))
				}
				if tot.Pulls == 0 {
					return 0
				}
				return tot.RewardSum / float64(tot.Pulls)
			}
			for arm, tot := range sampleTotals() {
				if got[arm].Pulls != int(tot.Pulls) || math.Abs(got[arm].Mean-wantMean(arm)) > 1e-12 {
					t.Errorf("arm %d = %+v, want %d pulls and mean %v", arm, got[arm], tot.Pulls, wantMean(arm))
				}
			}
		})
	}
}

// Restoring the totals of a trained policy into a fresh one gives it the same
// beliefs, so a restart loses nothing.
func TestRestoreReproducesTrainedPolicy(t *testing.T) {
	for _, spec := range PolicyNames() {
		t.Run(spec, func(t *testing.T) {
			trained := mustSnapshot(t, spec, 5)
			sums := make([]float64, 4)
			pulls := make([]int64, 4)
			for i := range 300 {
				arm := trained.Select()
				reward := float64((i + arm) % 2)
				trained.Update(arm, reward)
				sums[arm] += reward
				pulls[arm]++
			}
			totals := make([]ArmTotals, 4)
			for arm := range totals {
				totals[arm] = ArmTotals{Arm: arm, Pulls: pulls[arm], RewardSum: sums[arm]}
			}

			fresh := mustSnapshot(t, spec, 99)
			if err := fresh.Restore(totals); err != nil {
				t.Fatal(err)
			}
			a, b := trained.Snapshot(), fresh.Snapshot()
			for arm := range a {
				if a[arm].Pulls != b[arm].Pulls || math.Abs(a[arm].Mean-b[arm].Mean) > 1e-9 {
					t.Errorf("arm %d: trained %+v, restored %+v", arm, a[arm], b[arm])
				}
			}
		})
	}
}

func TestRestoreRejectsBadTotals(t *testing.T) {
	tests := []struct {
		name   string
		totals []ArmTotals
	}{
		{"too few arms", sampleTotals()[:2]},
		{"too many arms", append(sampleTotals(), ArmTotals{Arm: 4})},
		{"arms out of order", []ArmTotals{{Arm: 1}, {Arm: 0}, {Arm: 2}}},
		{"negative pulls", []ArmTotals{{Arm: 0}, {Arm: 1, Pulls: -1}, {Arm: 2}}},
		{"sum above pulls", []ArmTotals{{Arm: 0}, {Arm: 1, Pulls: 2, RewardSum: 3}, {Arm: 2}}},
		{"negative sum", []ArmTotals{{Arm: 0, RewardSum: -1}, {Arm: 1}, {Arm: 2}}},
		{"nan sum", []ArmTotals{{Arm: 0, Pulls: 1, RewardSum: math.NaN()}, {Arm: 1}, {Arm: 2}}},
	}
	for _, spec := range PolicyNames() {
		for _, tc := range tests {
			t.Run(spec+"/"+tc.name, func(t *testing.T) {
				pol, _ := NewPolicy(spec, 3, NewRNG(1, StreamPolicy))
				before := pol.(SnapshotPolicy).Snapshot()
				err := pol.(SnapshotPolicy).Restore(tc.totals)
				if !errors.Is(err, ErrBadTotals) {
					t.Fatalf("error = %v, want ErrBadTotals", err)
				}
				after := pol.(SnapshotPolicy).Snapshot()
				for arm := range before {
					if before[arm] != after[arm] {
						t.Errorf("a rejected Restore changed the policy: %+v -> %+v", before[arm], after[arm])
					}
				}
			})
		}
	}
}
