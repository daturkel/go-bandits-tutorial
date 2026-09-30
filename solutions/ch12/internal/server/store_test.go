package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"banditlab/internal/bandit"
	"banditlab/internal/store"
	"banditlab/internal/store/storetest"
)

// clock is a settable time source for the in-memory store.
type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newClock() *clock { return &clock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)} }

func TestRewardAfterTTLIsRejected(t *testing.T) {
	clk := newClock()
	st := store.NewMemory(3, store.WithClock(clk.now))
	h := New(newPolicy(t), st, quietLogger(), WithPendingTTL(time.Minute)).Handler()

	id := decode[selectResponse](t, do(h, "POST", "/select", "")).RequestID
	clk.advance(2 * time.Minute)

	rec := do(h, "POST", "/reward", `{"request_id":"`+id+`","reward":1}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("late reward: status = %d, want 404", rec.Code)
	}
	if got := decode[errorBody](t, rec).Error; got != store.ErrUnknownID.Error() {
		t.Errorf("message = %q", got)
	}
	if stats := decode[statsResponse](t, do(h, "GET", "/stats", "")); stats.Updates != 0 {
		t.Errorf("a late reward reached the policy: %+v", stats)
	}
}

func TestRewardWithinTTLIsAccepted(t *testing.T) {
	clk := newClock()
	st := store.NewMemory(3, store.WithClock(clk.now))
	h := New(newPolicy(t), st, quietLogger(), WithPendingTTL(time.Minute)).Handler()

	id := decode[selectResponse](t, do(h, "POST", "/select", "")).RequestID
	clk.advance(59 * time.Second)
	if rec := do(h, "POST", "/reward", `{"request_id":"`+id+`","reward":1}`); rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

// brokenStore fails every call that needs the database.
type brokenStore struct{ store.Store }

var errDown = errors.New("connection refused")

func (brokenStore) AddPending(context.Context, string, int, time.Duration) error { return errDown }
func (brokenStore) Reward(context.Context, string, float64) (int, error)         { return 0, errDown }
func (brokenStore) Ping(context.Context) error                                   { return errDown }

func TestStoreOutageIsA503WithoutLeakingDetails(t *testing.T) {
	h := New(newPolicy(t), brokenStore{store.NewMemory(3)}, quietLogger()).Handler()

	for _, tc := range []struct{ method, path, body string }{
		{"POST", "/select", ""},
		{"POST", "/reward", `{"request_id":"x","reward":1}`},
		{"GET", "/healthz", ""},
	} {
		rec := do(h, tc.method, tc.path, tc.body)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s %s: status = %d, want 503", tc.method, tc.path, rec.Code)
		}
		if body := rec.Body.String(); strings.Contains(body, "connection refused") {
			t.Errorf("%s %s leaks the driver error: %s", tc.method, tc.path, body)
		}
	}
}

func TestSyncRebuildsThePolicy(t *testing.T) {
	st := store.NewMemory(3)
	// A previous run recorded 20 rewards of 1 on arm 2 and 20 of 0 on arm 0.
	ctx := context.Background()
	for i := range 40 {
		id := string(rune('a'+i/26)) + string(rune('a'+i%26))
		arm, reward := 2, 1.0
		if i%2 == 1 {
			arm, reward = 0, 0
		}
		if err := st.AddPending(ctx, id, arm, time.Hour); err != nil {
			t.Fatal(err)
		}
		if _, err := st.Reward(ctx, id, reward); err != nil {
			t.Fatal(err)
		}
	}

	pol, err := bandit.NewSnapshotPolicy("thompson", 3, bandit.NewRNG(1, bandit.StreamPolicy))
	if err != nil {
		t.Fatal(err)
	}
	s := New(pol, st, quietLogger())
	if err := s.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	stats := decode[statsResponse](t, do(s.Handler(), "GET", "/stats", ""))
	if stats.Arms[2].Pulls != 20 || stats.Arms[2].Mean < 0.9 || stats.Arms[0].Mean > 0.1 {
		t.Errorf("restored arms = %+v", stats.Arms)
	}
	// And it acts on what it learned: with 20 clean wins, arm 2 dominates.
	picks := map[int]int{}
	for range 200 {
		picks[decode[selectResponse](t, do(s.Handler(), "POST", "/select", "")).Arm]++
	}
	if picks[2] < 150 {
		t.Errorf("restored thompson policy chose arm 2 only %d/200 times: %v", picks[2], picks)
	}
}

func TestSyncFailsOnMismatchedArms(t *testing.T) {
	s := New(newPolicy(t), store.NewMemory(5), quietLogger()) // policy has 3 arms, store 5
	if err := s.Sync(context.Background()); !errors.Is(err, bandit.ErrBadTotals) {
		t.Errorf("Restore = %v, want ErrBadTotals", err)
	}
}

func TestExpireLoopCountsSilenceAsFailure(t *testing.T) {
	clk := newClock()
	st := store.NewMemory(3, store.WithClock(clk.now))
	var logs bytes.Buffer
	s := New(newPolicy(t), st, slog.New(slog.NewTextHandler(&logs, nil)), WithPendingTTL(time.Minute))
	h := s.Handler()

	for range 3 {
		do(h, "POST", "/select", "") // never rewarded
	}
	clk.advance(2 * time.Minute)

	zero := 0.0
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.ExpireLoop(ctx, 5*time.Millisecond, &zero)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		stats := decode[statsResponse](t, do(h, "GET", "/stats", ""))
		pulls := 0
		for _, a := range stats.Arms {
			pulls += a.Pulls
		}
		if pulls == 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expired selections were never counted: %+v", stats)
		}
		time.Sleep(5 * time.Millisecond)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ExpireLoop did not stop when its context was cancelled")
	}
	// (Reading the log is safe now: the loop has finished.)
	if out := logs.String(); !strings.Contains(out, "count=3") || !strings.Contains(out, "counted_as=0") {
		t.Errorf("expiry log line should show count=3 counted_as=0, got: %s", out)
	}
	// The store agrees with the policy.
	totals, _ := st.Totals(context.Background())
	var pulls int64
	for _, a := range totals {
		pulls += a.Pulls
	}
	if pulls != 3 {
		t.Errorf("store counted %d pulls, want 3", pulls)
	}
}

// With a real database: a server restarts and keeps what it learned, including
// selections that were still waiting for a reward.
func TestServerSurvivesRestartWithPostgres(t *testing.T) {
	url := storetest.DatabaseURL(t)
	ctx := context.Background()
	schema := storetest.NewSchema(t, url)

	open := func() *Server {
		st, err := store.OpenPostgres(ctx, url, 3, store.WithSchema(schema))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { st.Close() })
		pol, err := bandit.NewSnapshotPolicy("thompson", 3, bandit.NewRNG(1, bandit.StreamPolicy))
		if err != nil {
			t.Fatal(err)
		}
		s := New(pol, st, quietLogger())
		if err := s.Sync(ctx); err != nil {
			t.Fatal(err)
		}
		return s
	}

	first := open().Handler()
	var waiting string
	for i := range 30 {
		sel := decode[selectResponse](t, do(first, "POST", "/select", ""))
		if i == 29 {
			waiting = sel.RequestID // leave the last one unrewarded
			break
		}
		reward := "0"
		if sel.Arm == 1 {
			reward = "1"
		}
		if rec := do(first, "POST", "/reward", `{"request_id":"`+sel.RequestID+`","reward":`+reward+`}`); rec.Code != http.StatusNoContent {
			t.Fatalf("reward status = %d: %s", rec.Code, rec.Body.String())
		}
	}
	before := decode[statsResponse](t, do(first, "GET", "/stats", ""))

	second := open().Handler() // a fresh process: new policy, same database
	after := decode[statsResponse](t, do(second, "GET", "/stats", ""))
	for arm := range 3 {
		if before.Arms[arm].Pulls != after.Arms[arm].Pulls {
			t.Errorf("arm %d pulls: before restart %d, after %d", arm, before.Arms[arm].Pulls, after.Arms[arm].Pulls)
		}
	}
	if rec := do(second, "POST", "/reward", `{"request_id":"`+waiting+`","reward":1}`); rec.Code != http.StatusNoContent {
		t.Errorf("late reward for a pre-restart selection: status = %d", rec.Code)
	}
}

// Two replicas on one store: a reward that one handles reaches the other's
// policy only when the other syncs, and pending selections come along too.
func TestSyncSharesWhatOtherReplicasLearned(t *testing.T) {
	st := store.NewMemory(3)
	ctx := context.Background()
	a := New(newPolicy(t), st, quietLogger())
	b := New(newPolicy(t), st, quietLogger())
	ha, hb := a.Handler(), b.Handler()

	for range 5 {
		id := decode[selectResponse](t, do(ha, "POST", "/select", "")).RequestID
		do(ha, "POST", "/reward", `{"request_id":"`+id+`","reward":1}`)
	}
	waiting := decode[selectResponse](t, do(ha, "POST", "/select", "")) // never rewarded

	before := decode[statsResponse](t, do(hb, "GET", "/stats", ""))
	if before.Updates != 0 {
		t.Fatalf("replica b knows about %d updates before syncing, want 0", before.Updates)
	}
	if err := b.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	after := decode[statsResponse](t, do(hb, "GET", "/stats", ""))
	pulls, pending := 0, 0
	for _, arm := range after.Arms {
		pulls += arm.Pulls
		pending += arm.Pending
	}
	if pulls != 5 || pending != 1 {
		t.Errorf("after Sync replica b sees %d pulls and %d pending, want 5 and 1 (arms: %+v)", pulls, pending, after.Arms)
	}
	if after.Arms[waiting.Arm].Pending != 1 {
		t.Errorf("the unrewarded selection on arm %d is not pending on b: %+v", waiting.Arm, after.Arms)
	}
}

// If recording a selection fails, the reserved slot is released; otherwise a
// storage outage would slowly convince UCB1 that every arm is over-sampled.
func TestFailedSelectionReleasesItsPendingSlot(t *testing.T) {
	pol := newPolicy(t)
	s := New(pol, brokenStore{store.NewMemory(3)}, quietLogger())
	for range 3 {
		if rec := do(s.Handler(), "POST", "/select", ""); rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
	}
	for _, arm := range pol.Snapshot() {
		if arm.Pending != 0 {
			t.Errorf("arm %d still has %d pending after failed selections", arm.Arm, arm.Pending)
		}
	}
}
