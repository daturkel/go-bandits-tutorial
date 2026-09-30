package store_test

import (
	"context"
	"testing"
	"time"

	"banditlab/internal/store"
	"banditlab/internal/store/storetest"
)

func TestMemoryConformance(t *testing.T) {
	storetest.Run(t, func(t *testing.T, arms int) store.Store {
		return store.NewMemory(arms)
	})
}

// The in-memory store can also be driven by a fake clock, which makes expiry
// exact instead of sleep-based.
func TestMemoryFakeClock(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m := store.NewMemory(2, store.WithClock(func() time.Time { return now }))
	ctx := context.Background()

	if err := m.AddPending(ctx, "a", 0, time.Minute); err != nil {
		t.Fatal(err)
	}
	now = now.Add(59 * time.Second)
	if _, err := m.Reward(ctx, "a", 1); err != nil {
		t.Fatalf("Reward just before expiry = %v", err)
	}

	if err := m.AddPending(ctx, "b", 1, time.Minute); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute) // exactly at the deadline: expired
	if _, err := m.Reward(ctx, "b", 1); err == nil {
		t.Error("Reward at the exact expiry time succeeded")
	}
}
