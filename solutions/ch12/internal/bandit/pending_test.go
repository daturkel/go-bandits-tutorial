package bandit

import (
	"context"
	"errors"
	"math"
	"testing"
)

var (
	_ PendingTracker = (*UCB1)(nil)
	_ PendingTracker = (*Locked)(nil)
)

// The behaviour that motivated pending-awareness: requests that arrive before
// any reward is back.
func TestUCB1SpreadsSelectionsWhileRewardsAreOutstanding(t *testing.T) {
	aware, _ := NewUCB1(4)
	naive, _ := NewUCB1(4, IgnorePending())

	var gotAware, gotNaive []int
	for range 4 {
		gotAware = append(gotAware, aware.Select())
		gotNaive = append(gotNaive, naive.Select())
	}
	for i, want := range []int{0, 1, 2, 3} {
		if gotAware[i] != want {
			t.Errorf("pending-aware UCB1 selections = %v, want each untried arm once: 0 1 2 3", gotAware)
			break
		}
	}
	for _, arm := range gotNaive {
		if arm != 0 {
			t.Errorf("textbook UCB1 selections = %v, want arm 0 repeated (nothing has changed)", gotNaive)
			break
		}
	}
}

func TestUCB1PendingBonusShrinksAsSelectionsPileUp(t *testing.T) {
	p, _ := NewUCB1(2)
	// Both arms have a full history; arm 1 looks a little better.
	if err := p.Restore([]ArmTotals{{Arm: 0, Pulls: 100, RewardSum: 50}, {Arm: 1, Pulls: 100, RewardSum: 55}}); err != nil {
		t.Fatal(err)
	}
	// With nothing pending, arm 1 wins every time it is asked and nothing changes.
	first := p.Select()
	if first != 1 {
		t.Fatalf("first selection = %d, want the better arm", first)
	}
	// Each further unanswered selection of arm 1 makes it look better explored,
	// so at some point the policy tries arm 0 instead of piling on.
	sawOther := false
	for range 200 {
		if p.Select() == 0 {
			sawOther = true
			break
		}
	}
	if !sawOther {
		t.Error("200 unanswered selections all went to the same arm")
	}
}

func TestUCB1UpdateAndAbandonReleasePending(t *testing.T) {
	p, _ := NewUCB1(3)
	a, b, c := p.Select(), p.Select(), p.Select() // 0, 1, 2 all pending
	stat := func() (pending int) {
		for _, s := range p.Snapshot() {
			pending += s.Pending
		}
		return pending
	}
	if stat() != 3 {
		t.Fatalf("pending = %d, want 3", stat())
	}
	p.Update(a, 1)
	if stat() != 2 {
		t.Errorf("after Update pending = %d, want 2", stat())
	}
	p.Abandon(b)
	if stat() != 1 {
		t.Errorf("after Abandon pending = %d, want 1", stat())
	}
	// Releasing more than was reserved never goes negative: a reward may
	// belong to a selection another replica made.
	p.Abandon(b)
	p.Update(b, 0)
	if got := p.Snapshot()[b].Pending; got != 0 {
		t.Errorf("pending for arm %d = %d, want 0", b, got)
	}
	_ = c
}

func TestUCB1SetPending(t *testing.T) {
	p, _ := NewUCB1(3)
	if err := p.SetPending([]int{2, 0, 5}); err != nil {
		t.Fatal(err)
	}
	snap := p.Snapshot()
	if snap[0].Pending != 2 || snap[1].Pending != 0 || snap[2].Pending != 5 {
		t.Errorf("Snapshot pending = %+v", snap)
	}
	for _, bad := range [][]int{{1, 2}, {1, 2, 3, 4}, {0, -1, 0}} {
		if err := p.SetPending(bad); !errors.Is(err, ErrBadTotals) {
			t.Errorf("SetPending(%v) = %v, want ErrBadTotals", bad, err)
		}
	}
	if p.Snapshot()[2].Pending != 5 {
		t.Error("a rejected SetPending changed the state")
	}
}

func TestPolicySpecUCB1Naive(t *testing.T) {
	pol, err := NewPolicy("ucb1:naive", 3, NewRNG(1, StreamPolicy))
	if err != nil || pol.Name() != "ucb1-naive" {
		t.Fatalf("NewPolicy(ucb1:naive) = (%v, %v)", pol, err)
	}
	if _, err := NewPolicy("ucb1:fast", 3, NewRNG(1, StreamPolicy)); err == nil {
		t.Error("an unknown ucb1 argument was accepted")
	}
}

func TestLockedForwardsPendingCalls(t *testing.T) {
	u, _ := NewUCB1(2)
	l := NewLocked(u)
	l.Select()
	l.Select()
	if err := l.SetPending([]int{3, 1}); err != nil {
		t.Fatal(err)
	}
	l.Abandon(0)
	if snap := l.Snapshot(); snap[0].Pending != 2 || snap[1].Pending != 1 {
		t.Errorf("pending = %+v, want [2 1]", snap)
	}
	// Policies without pending tracking accept the calls and ignore them.
	th, _ := NewThompson(2, NewRNG(1, StreamPolicy))
	lt := NewLocked(th)
	lt.Abandon(1)
	if err := lt.SetPending([]int{1, 1}); err != nil {
		t.Errorf("SetPending on Thompson = %v, want nil", err)
	}
}

// ---- delayed feedback in the simulator ----

func TestRunDelayedWithZeroDelayMatchesRun(t *testing.T) {
	probs := []float64{0.2, 0.5, 0.8}
	mk := func() (*Env, Policy) {
		env, _ := NewEnv(probs, NewRNG(3, StreamEnv))
		pol, _ := NewPolicy("thompson", 3, NewRNG(3, StreamPolicy))
		return env, pol
	}
	e1, p1 := mk()
	a, _ := Run(context.Background(), e1, p1, 500)
	e2, p2 := mk()
	b, _ := RunDelayed(context.Background(), e2, p2, 500, 0)
	if a.Regret != b.Regret || a.TotalReward != b.TotalReward {
		t.Errorf("Run and RunDelayed(0) differ: %v vs %v", a.Regret, b.Regret)
	}
}

func TestRunDelayedRejectsNegativeDelay(t *testing.T) {
	env, _ := NewEnv([]float64{0.5, 0.5}, NewRNG(1, StreamEnv))
	pol, _ := NewUCB1(2)
	if _, err := RunDelayed(context.Background(), env, pol, 10, -1); err == nil {
		t.Error("negative delay accepted")
	}
}

// recorder checks when rewards are delivered.
type recorder struct {
	Policy
	selects, updates int
	maxOutstanding   int
}

func (r *recorder) Select() int {
	r.selects++
	if out := r.selects - r.updates; out > r.maxOutstanding {
		r.maxOutstanding = out
	}
	return r.Policy.Select()
}

func (r *recorder) Update(arm int, reward float64) {
	r.updates++
	r.Policy.Update(arm, reward)
}

func TestRunDelayedDelaysRewards(t *testing.T) {
	env, _ := NewEnv([]float64{0.5, 0.5}, NewRNG(1, StreamEnv))
	inner, _ := NewUCB1(2)
	r := &recorder{Policy: inner}
	if _, err := RunDelayed(context.Background(), env, r, 100, 7); err != nil {
		t.Fatal(err)
	}
	if r.maxOutstanding != 8 { // the 7 undelivered rewards plus the selection just made
		t.Errorf("at most %d selections outstanding, want 8", r.maxOutstanding)
	}
	if r.updates != 100-7 {
		t.Errorf("%d rewards delivered, want 93 (the last 7 are discarded)", r.updates)
	}
}

// The effect of the fix, end to end. With a feedback delay of 200 steps, the
// textbook algorithm spends its first 200 choices on one arm, and its initial
// tour of the arms takes 200 steps per arm.
func TestPendingAwarenessReducesRegretUnderDelay(t *testing.T) {
	probs, _ := Scenario("easy")
	const seeds = 20
	regret := map[string]float64{}
	for seed := uint64(1); seed <= seeds; seed++ {
		for _, spec := range []string{"ucb1:naive", "ucb1"} {
			env, _ := NewEnv(probs, NewRNG(seed, StreamEnv))
			pol, _ := NewPolicy(spec, 3, NewRNG(seed, StreamPolicy))
			res, err := RunDelayed(context.Background(), env, pol, 3000, 200)
			if err != nil {
				t.Fatal(err)
			}
			regret[spec] += res.Regret / seeds
		}
	}
	if regret["ucb1"] >= 0.6*regret["ucb1:naive"] {
		t.Errorf("mean regret: pending-aware %.0f, naive %.0f; want the fix to cut it by at least 40%%",
			regret["ucb1"], regret["ucb1:naive"])
	}
}

// ---- merging ----

func randomTotals(rng interface {
	IntN(int) int
	Float64() float64
}, arms int) []ArmTotals {
	out := make([]ArmTotals, arms)
	for i := range out {
		pulls := int64(rng.IntN(1000))
		out[i] = ArmTotals{Arm: i, Pulls: pulls, RewardSum: rng.Float64() * float64(pulls)}
	}
	return out
}

func equalTotals(a, b []ArmTotals) bool {
	for i := range a {
		if a[i].Arm != b[i].Arm || a[i].Pulls != b[i].Pulls || math.Abs(a[i].RewardSum-b[i].RewardSum) > 1e-9 {
			return false
		}
	}
	return len(a) == len(b)
}

func TestMergeLaws(t *testing.T) {
	rng := NewRNG(1, StreamEnv)
	zero := make([]ArmTotals, 5)
	for i := range zero {
		zero[i].Arm = i
	}
	for range 200 {
		a, b, c := randomTotals(rng, 5), randomTotals(rng, 5), randomTotals(rng, 5)
		ab, _ := Merge(a, b)
		ba, _ := Merge(b, a)
		if !equalTotals(ab, ba) {
			t.Fatal("Merge is not commutative")
		}
		abc1, _ := Merge(ab, c)
		bc, _ := Merge(b, c)
		abc2, _ := Merge(a, bc)
		if !equalTotals(abc1, abc2) {
			t.Fatal("Merge is not associative")
		}
		id, _ := Merge(a, zero)
		if !equalTotals(id, a) {
			t.Fatal("zero totals are not an identity")
		}
	}
}

func TestMergeRejectsMismatches(t *testing.T) {
	a := []ArmTotals{{Arm: 0}, {Arm: 1}}
	if _, err := Merge(a, a[:1]); !errors.Is(err, ErrBadTotals) {
		t.Errorf("different lengths: %v", err)
	}
	if _, err := Merge(a, []ArmTotals{{Arm: 1}, {Arm: 0}}); !errors.Is(err, ErrBadTotals) {
		t.Errorf("wrong arm order: %v", err)
	}
}

// Two replicas each learn from half of the data. Adding their totals gives
// exactly the beliefs of one policy that saw everything, because a Beta
// posterior depends only on the counts, not on the order of the observations.
func TestMergedTotalsGiveTheSameThompsonPosterior(t *testing.T) {
	type obs struct {
		arm    int
		reward float64
	}
	rng := NewRNG(7, StreamEnv)
	var data []obs
	for range 500 {
		data = append(data, obs{arm: rng.IntN(3), reward: float64(rng.IntN(2))})
	}

	totalsOf := func(part []obs) []ArmTotals {
		out := make([]ArmTotals, 3)
		for i := range out {
			out[i].Arm = i
		}
		for _, o := range part {
			out[o.arm].Pulls++
			out[o.arm].RewardSum += o.reward
		}
		return out
	}

	all, _ := NewThompson(3, NewRNG(1, StreamPolicy))
	for _, o := range data {
		all.Update(o.arm, o.reward)
	}

	merged, err := Merge(totalsOf(data[:200]), totalsOf(data[200:]))
	if err != nil {
		t.Fatal(err)
	}
	fromMerge, _ := NewThompson(3, NewRNG(2, StreamPolicy))
	if err := fromMerge.Restore(merged); err != nil {
		t.Fatal(err)
	}
	a, b := all.Snapshot(), fromMerge.Snapshot()
	for arm := range a {
		if a[arm].Pulls != b[arm].Pulls || math.Abs(a[arm].Mean-b[arm].Mean) > 1e-12 {
			t.Errorf("arm %d: one policy believes %+v, merged replicas believe %+v", arm, a[arm], b[arm])
		}
	}
}
