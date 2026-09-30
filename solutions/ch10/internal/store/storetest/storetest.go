// Package storetest holds tests that every store.Store implementation must
// pass. Each implementation's own test file calls Run with a constructor, so
// the in-memory and the PostgreSQL stores are held to exactly the same
// behaviour. (The standard library does the same in testing/fstest.)
package storetest

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"banditlab/internal/store"
)

// New builds a fresh, empty store with the given number of arms. It should
// arrange for the store to be closed when the test ends.
type New func(t *testing.T, arms int) store.Store

// Run runs the conformance tests against stores made by newStore.
func Run(t *testing.T, newStore New) {
	t.Run("reward updates totals", func(t *testing.T) { testReward(t, newStore(t, 3)) })
	t.Run("reward is applied once", func(t *testing.T) { testOnce(t, newStore(t, 3)) })
	t.Run("unknown id", func(t *testing.T) { testUnknown(t, newStore(t, 3)) })
	t.Run("duplicate selection id", func(t *testing.T) { testDuplicate(t, newStore(t, 3)) })
	t.Run("expired selections cannot be rewarded", func(t *testing.T) { testExpiredReward(t, newStore(t, 3)) })
	t.Run("expire without implicit reward", func(t *testing.T) { testExpire(t, newStore(t, 3), nil) })
	t.Run("expire with implicit reward", func(t *testing.T) {
		zero := 0.0
		testExpire(t, newStore(t, 3), &zero)
	})
	t.Run("concurrent rewards for one id", func(t *testing.T) { testConcurrentSameID(t, newStore(t, 3)) })
	t.Run("concurrent rewards for different ids", func(t *testing.T) { testConcurrentDifferentIDs(t, newStore(t, 3)) })
	t.Run("ping", func(t *testing.T) {
		if err := newStore(t, 1).Ping(context.Background()); err != nil {
			t.Errorf("Ping = %v", err)
		}
	})
}

var ctx = context.Background()

func totals(t *testing.T, s store.Store) (pulls []int64, sums []float64) {
	t.Helper()
	all, err := s.Totals(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for i, a := range all {
		if a.Arm != i {
			t.Fatalf("totals out of order: entry %d is arm %d", i, a.Arm)
		}
		pulls = append(pulls, a.Pulls)
		sums = append(sums, a.RewardSum)
	}
	return pulls, sums
}

func testReward(t *testing.T, s store.Store) {
	if pulls, _ := totals(t, s); len(pulls) != 3 {
		t.Fatalf("new store has %d arms, want 3", len(pulls))
	}
	if err := s.AddPending(ctx, "a", 1, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := s.AddPending(ctx, "b", 2, time.Minute); err != nil {
		t.Fatal(err)
	}
	arm, err := s.Reward(ctx, "a", 0.75)
	if err != nil || arm != 1 {
		t.Fatalf("Reward(a) = (%d, %v), want (1, nil)", arm, err)
	}
	if _, err := s.Reward(ctx, "b", 0); err != nil {
		t.Fatal(err)
	}
	pulls, sums := totals(t, s)
	if pulls[0] != 0 || pulls[1] != 1 || pulls[2] != 1 || sums[1] != 0.75 || sums[2] != 0 {
		t.Errorf("totals = %v / %v", pulls, sums)
	}
}

func testOnce(t *testing.T, s store.Store) {
	if err := s.AddPending(ctx, "a", 0, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Reward(ctx, "a", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Reward(ctx, "a", 1); !errors.Is(err, store.ErrAlreadyRewarded) {
		t.Errorf("second Reward = %v, want ErrAlreadyRewarded", err)
	}
	if pulls, sums := totals(t, s); pulls[0] != 1 || sums[0] != 1 {
		t.Errorf("totals changed by the rejected reward: %v / %v", pulls, sums)
	}
}

func testUnknown(t *testing.T, s store.Store) {
	if _, err := s.Reward(ctx, "nope", 1); !errors.Is(err, store.ErrUnknownID) {
		t.Errorf("Reward(nope) = %v, want ErrUnknownID", err)
	}
	if pulls, _ := totals(t, s); pulls[0]+pulls[1]+pulls[2] != 0 {
		t.Errorf("totals changed: %v", pulls)
	}
}

func testDuplicate(t *testing.T, s store.Store) {
	if err := s.AddPending(ctx, "a", 0, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := s.AddPending(ctx, "a", 1, time.Minute); err == nil {
		t.Error("adding the same id twice succeeded")
	}
	if err := s.AddPending(ctx, "z", 99, time.Minute); err == nil {
		t.Error("adding an id for a nonexistent arm succeeded")
	}
}

func testExpiredReward(t *testing.T, s store.Store) {
	if err := s.AddPending(ctx, "a", 0, 30*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)
	if _, err := s.Reward(ctx, "a", 1); !errors.Is(err, store.ErrUnknownID) {
		t.Errorf("Reward after expiry = %v, want ErrUnknownID", err)
	}
	if pulls, _ := totals(t, s); pulls[0] != 0 {
		t.Errorf("an expired reward changed the totals: %v", pulls)
	}
}

// Three selections expire: two unrewarded (arms 0 and 0), one rewarded (arm 1).
// A fourth is still valid and must be left alone.
func testExpire(t *testing.T, s store.Store, implicit *float64) {
	for id, arm := range map[string]int{"u1": 0, "u2": 0, "r1": 1} {
		if err := s.AddPending(ctx, id, arm, 30*time.Millisecond); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Reward(ctx, "r1", 1); err != nil {
		t.Fatal(err)
	}
	if err := s.AddPending(ctx, "live", 2, time.Minute); err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)

	expired, err := s.Expire(ctx, implicit)
	if err != nil {
		t.Fatal(err)
	}
	if len(expired) != 1 || expired[0] != 2 {
		t.Errorf("Expire reported %v, want only arm 0 with 2 unrewarded selections", expired)
	}

	pulls, _ := totals(t, s)
	wantArm0 := int64(0)
	if implicit != nil {
		wantArm0 = 2
	}
	if pulls[0] != wantArm0 || pulls[1] != 1 || pulls[2] != 0 {
		t.Errorf("pulls after Expire = %v, want [%d 1 0]", pulls, wantArm0)
	}

	// The valid selection survived, and the expired ones are really gone.
	if _, err := s.Reward(ctx, "live", 1); err != nil {
		t.Errorf("Reward(live) = %v after Expire", err)
	}
	if _, err := s.Reward(ctx, "u1", 1); !errors.Is(err, store.ErrUnknownID) {
		t.Errorf("Reward(u1) = %v, want ErrUnknownID", err)
	}
	if again, err := s.Expire(ctx, implicit); err != nil || len(again) != 0 {
		t.Errorf("second Expire = (%v, %v), want nothing", again, err)
	}
}

// Many clients retry the same reward at once. Exactly one may win.
func testConcurrentSameID(t *testing.T, s store.Store) {
	if err := s.AddPending(ctx, "a", 2, time.Minute); err != nil {
		t.Fatal(err)
	}
	const callers = 20
	var wins, dups atomic.Int32
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch _, err := s.Reward(ctx, "a", 1); {
			case err == nil:
				wins.Add(1)
			case errors.Is(err, store.ErrAlreadyRewarded):
				dups.Add(1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 || dups.Load() != callers-1 {
		t.Errorf("%d successes and %d duplicates, want 1 and %d", wins.Load(), dups.Load(), callers-1)
	}
	if pulls, sums := totals(t, s); pulls[2] != 1 || sums[2] != 1 {
		t.Errorf("totals = %v / %v, want the reward counted exactly once", pulls, sums)
	}
}

func testConcurrentDifferentIDs(t *testing.T, s store.Store) {
	const n = 60
	ids := make([]string, n)
	for i := range ids {
		ids[i] = string(rune('A'+i/26)) + string(rune('a'+i%26))
		if err := s.AddPending(ctx, ids[i], i%3, time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Reward(ctx, id, 0.5); err != nil {
				t.Errorf("Reward(%s) = %v", id, err)
			}
		}()
	}
	wg.Wait()
	pulls, sums := totals(t, s)
	for arm := range 3 {
		if pulls[arm] != n/3 || sums[arm] != float64(n/3)*0.5 {
			t.Errorf("arm %d: pulls %d sum %v, want %d and %v", arm, pulls[arm], sums[arm], n/3, float64(n/3)*0.5)
		}
	}
}
