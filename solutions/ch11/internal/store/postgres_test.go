package store_test

import (
	"context"
	"testing"
	"time"

	"banditlab/internal/store"
	"banditlab/internal/store/storetest"
)

func TestPostgresConformance(t *testing.T) {
	url := storetest.DatabaseURL(t)
	storetest.Run(t, func(t *testing.T, arms int) store.Store {
		s, err := store.OpenPostgres(context.Background(), url, arms, store.WithSchema(storetest.NewSchema(t, url)))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { s.Close() })
		return s
	})
}

// State outlives the process: close the store, open a new one on the same
// schema, and the totals are still there.
func TestPostgresPersistsAcrossRestarts(t *testing.T) {
	url := storetest.DatabaseURL(t)
	ctx := context.Background()
	schema := storetest.NewSchema(t, url)

	first, err := store.OpenPostgres(ctx, url, 3, store.WithSchema(schema))
	if err != nil {
		t.Fatal(err)
	}
	if err := first.AddPending(ctx, "a", 2, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := first.Reward(ctx, "a", 0.5); err != nil {
		t.Fatal(err)
	}
	// A selection that is still waiting when the "process" exits.
	if err := first.AddPending(ctx, "waiting", 1, time.Minute); err != nil {
		t.Fatal(err)
	}
	first.Close()

	second, err := store.OpenPostgres(ctx, url, 3, store.WithSchema(schema))
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	got, err := second.Totals(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got[2].Pulls != 1 || got[2].RewardSum != 0.5 {
		t.Errorf("totals after restart = %+v", got)
	}
	if _, err := second.Reward(ctx, "waiting", 1); err != nil {
		t.Errorf("a selection made before the restart could not be rewarded: %v", err)
	}
}

func TestPostgresRefusesToDropArms(t *testing.T) {
	url := storetest.DatabaseURL(t)
	ctx := context.Background()
	schema := storetest.NewSchema(t, url)

	s, err := store.OpenPostgres(ctx, url, 5, store.WithSchema(schema))
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	if _, err := store.OpenPostgres(ctx, url, 3, store.WithSchema(schema)); err == nil {
		t.Error("opening a 5-arm database as 3 arms succeeded")
	}
	// Growing is fine: new arms start empty.
	grown, err := store.OpenPostgres(ctx, url, 7, store.WithSchema(schema))
	if err != nil {
		t.Fatalf("growing to 7 arms: %v", err)
	}
	grown.Close()
}

func TestPostgresBadURL(t *testing.T) {
	if _, err := store.OpenPostgres(context.Background(), "not a url", 3); err == nil {
		t.Error("expected an error for a malformed URL")
	}
}
