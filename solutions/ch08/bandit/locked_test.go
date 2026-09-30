package bandit

import (
	"slices"
	"sync"
	"testing"
)

// Compile-time checks.
var (
	_ SnapshotPolicy = (*EpsilonGreedy)(nil)
	_ SnapshotPolicy = (*UCB1)(nil)
	_ SnapshotPolicy = (*Thompson)(nil)
	_ SnapshotPolicy = (*Locked)(nil)
)

func newLocked(t *testing.T, spec string) *Locked {
	t.Helper()
	pol, err := NewPolicy(spec, 4, NewRNG(1, StreamPolicy))
	if err != nil {
		t.Fatal(err)
	}
	return NewLocked(pol.(SnapshotPolicy))
}

// Many goroutines share one policy. Which arm each one gets depends on
// scheduling, so the test checks invariants that must hold regardless: no
// update is lost, and the counters agree. Run it with -race.
func TestLockedConcurrentUse(t *testing.T) {
	for _, spec := range PolicyNames() {
		t.Run(spec, func(t *testing.T) {
			l := newLocked(t, spec)
			const workers, perWorker = 8, 500

			var wg sync.WaitGroup
			for range workers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for range perWorker {
						arm := l.Select()
						l.Update(arm, 1)
					}
				}()
			}
			wg.Wait()

			total := 0
			for _, s := range l.Snapshot() {
				total += s.Pulls
			}
			if want := workers * perWorker; total != want {
				t.Errorf("snapshot counts %d pulls, want %d", total, want)
			}
			sel, upd := l.Calls()
			if sel != workers*perWorker || upd != workers*perWorker {
				t.Errorf("Calls() = (%d, %d), want (%d, %d)", sel, upd, workers*perWorker, workers*perWorker)
			}
		})
	}
}

// Wrapping must not change what a single goroutine sees.
func TestLockedMatchesBarePolicy(t *testing.T) {
	bare, _ := NewPolicy("thompson", 4, NewRNG(9, StreamPolicy))
	wrapped := NewLocked(mustSnapshot(t, "thompson", 9))
	for step := range 200 {
		a, b := bare.Select(), wrapped.Select()
		if a != b {
			t.Fatalf("step %d: bare chose %d, wrapped chose %d", step, a, b)
		}
		reward := float64(step % 2)
		bare.Update(a, reward)
		wrapped.Update(b, reward)
	}
}

func mustSnapshot(t *testing.T, spec string, seed uint64) SnapshotPolicy {
	t.Helper()
	pol, err := NewPolicy(spec, 4, NewRNG(seed, StreamPolicy))
	if err != nil {
		t.Fatal(err)
	}
	return pol.(SnapshotPolicy)
}

func TestSnapshotIsACopy(t *testing.T) {
	l := newLocked(t, "ucb1")
	l.Update(2, 1)
	snap := l.Snapshot()
	snap[2].Pulls = 999
	if got := l.Snapshot()[2].Pulls; got != 1 {
		t.Errorf("modifying a snapshot changed the policy: pulls = %d", got)
	}
}

func TestSnapshotValues(t *testing.T) {
	tests := []struct {
		spec     string
		wantPull int
		wantMean float64
	}{
		{"epsgreedy", 3, 2.0 / 3},
		{"ucb1", 3, 2.0 / 3},
		// Beta(1+2, 1+1) has mean 3/5.
		{"thompson", 3, 3.0 / 5},
	}
	for _, tc := range tests {
		t.Run(tc.spec, func(t *testing.T) {
			p := mustSnapshot(t, tc.spec, 1)
			p.Update(1, 1)
			p.Update(1, 1)
			p.Update(1, 0)
			got := p.Snapshot()[1]
			if got.Arm != 1 || got.Pulls != tc.wantPull || got.Mean < tc.wantMean-1e-9 || got.Mean > tc.wantMean+1e-9 {
				t.Errorf("Snapshot()[1] = %+v, want pulls %d mean %v", got, tc.wantPull, tc.wantMean)
			}
			if untouched := p.Snapshot()[0]; untouched.Pulls != 0 {
				t.Errorf("arm 0 = %+v, want no pulls", untouched)
			}
		})
	}
}

func TestSnapshotLength(t *testing.T) {
	l := newLocked(t, "epsgreedy")
	arms := make([]int, 0, 4)
	for _, s := range l.Snapshot() {
		arms = append(arms, s.Arm)
	}
	if !slices.Equal(arms, []int{0, 1, 2, 3}) {
		t.Errorf("arms = %v", arms)
	}
}
