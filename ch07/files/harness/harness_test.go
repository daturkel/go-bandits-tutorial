package harness

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"
	"testing"
	"time"
)

func testSpec() Spec {
	return Spec{
		Scenario: "easy",
		Probs:    []float64{0.2, 0.5, 0.8},
		Policies: []string{"epsgreedy:0.1", "ucb1", "thompson"},
		Steps:    500,
		Seeds:    8,
		BaseSeed: 1,
	}
}

func TestCompareShapes(t *testing.T) {
	rep, err := Compare(context.Background(), testSpec())
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Curves) != 3 {
		t.Fatalf("got %d curves, want 3", len(rep.Curves))
	}
	for _, c := range rep.Curves {
		if len(c.Mean) != 500 || len(c.StdErr) != 500 || len(c.Final) != 8 {
			t.Errorf("%s: bad lengths %d/%d/%d", c.Policy, len(c.Mean), len(c.StdErr), len(c.Final))
		}
		for i := 1; i < len(c.Mean); i++ {
			if c.Mean[i] < c.Mean[i-1] {
				t.Fatalf("%s: mean regret decreased at step %d", c.Policy, i)
			}
		}
	}
}

func TestCompareIsDeterministic(t *testing.T) {
	a, err := Compare(context.Background(), testSpec())
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Compare(context.Background(), testSpec())
	for i := range a.Curves {
		if a.Curves[i].Mean[499] != b.Curves[i].Mean[499] {
			t.Errorf("%s: repeat run differs", a.Curves[i].Policy)
		}
	}
}

func TestCompareRejectsBadInput(t *testing.T) {
	bad := testSpec()
	bad.Policies = []string{"nope"}
	if _, err := Compare(context.Background(), bad); err == nil {
		t.Error("unknown policy should fail")
	}
	bad = testSpec()
	bad.Seeds = 0
	if _, err := Compare(context.Background(), bad); err == nil {
		t.Error("zero seeds should fail")
	}
}

func TestWriteCSV(t *testing.T) {
	rep, _ := Compare(context.Background(), testSpec())
	var buf bytes.Buffer
	if err := rep.WriteCSV(&buf, 10); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if want := 1 + 3*10; len(rows) != want {
		t.Fatalf("got %d rows, want %d", len(rows), want)
	}
	if rows[1][0] != "1" || rows[10][0] != "500" {
		t.Errorf("first policy should span steps 1..500, got %s..%s", rows[1][0], rows[10][0])
	}
}

func TestWriteSVG(t *testing.T) {
	rep, _ := Compare(context.Background(), testSpec())
	var buf bytes.Buffer
	if err := rep.WriteSVG(&buf); err != nil {
		t.Fatal(err)
	}
	svg := buf.String()
	for _, want := range []string{"<svg", "</svg>", "epsilon-greedy(0.10)", "ucb1", "thompson", "<polyline"} {
		if !strings.Contains(svg, want) {
			t.Errorf("svg is missing %q", want)
		}
	}
	if strings.Contains(svg, "NaN") || strings.Contains(svg, "Inf") {
		t.Error("svg contains a non-finite coordinate")
	}
}

func TestNiceTicks(t *testing.T) {
	ticks := niceTicks(437, 5)
	if ticks[0] != 0 || ticks[len(ticks)-1] < 437 || math.IsNaN(ticks[1]) {
		t.Errorf("unexpected ticks %v", ticks)
	}
	if got := niceTicks(0, 5); len(got) != 2 {
		t.Errorf("zero max ticks = %v", got)
	}
}

// The number of workers changes how the runs are scheduled, never the report.
func TestCompareIsIndependentOfWorkers(t *testing.T) {
	var reports []*Report
	for _, workers := range []int{1, 2, 7} {
		spec := testSpec()
		spec.Workers = workers
		rep, err := Compare(context.Background(), spec)
		if err != nil {
			t.Fatal(err)
		}
		reports = append(reports, rep)
	}
	for _, rep := range reports[1:] {
		for i := range rep.Curves {
			for step := range rep.Curves[i].Mean {
				if rep.Curves[i].Mean[step] != reports[0].Curves[i].Mean[step] {
					t.Fatalf("%s: mean differs at step %d", rep.Curves[i].Policy, step)
				}
			}
		}
	}
}

// With several bad policies, the error names the first in spec order every
// time, however the goroutines are scheduled.
func TestCompareErrorIsDeterministic(t *testing.T) {
	spec := testSpec()
	spec.Policies = []string{"ucb1", "first-bad", "second-bad"}
	for range 20 {
		_, err := Compare(context.Background(), spec)
		if err == nil || !strings.Contains(err.Error(), "first-bad") {
			t.Fatalf("error = %v, want one naming first-bad", err)
		}
	}
}

func TestCompareProgress(t *testing.T) {
	spec := testSpec()
	var calls, lastDone, lastTotal int
	spec.OnProgress = func(done, total int) {
		calls++ // no lock: calls never overlap
		lastDone, lastTotal = done, total
	}
	if _, err := Compare(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	want := len(spec.Policies) * spec.Seeds
	if calls != want || lastDone != want || lastTotal != want {
		t.Errorf("calls %d, last %d/%d; want %d each", calls, lastDone, lastTotal, want)
	}
}

// bigSpec would take many seconds if it were allowed to finish.
func bigSpec() Spec {
	spec := testSpec()
	spec.Steps = 200_000
	spec.Seeds = 2000
	return spec
}

func TestCompareCancelled(t *testing.T) {
	before := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := Compare(ctx, bigSpec())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("took %v to notice the cancellation", elapsed)
	}
	waitForGoroutines(t, before)
}

func TestCompareTimeout(t *testing.T) {
	before := runtime.NumGoroutine()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, err := Compare(ctx, bigSpec())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context.DeadlineExceeded", err)
	}
	waitForGoroutines(t, before)
}

// waitForGoroutines fails the test if goroutines started since `before` do
// not exit. It polls, because exiting goroutines need a moment to be noticed.
func waitForGoroutines(t *testing.T, before int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<16)
			t.Fatalf("goroutine leak: %d running, %d before\n%s", runtime.NumGoroutine(), before, buf[:runtime.Stack(buf, true)])
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func BenchmarkCompare(b *testing.B) {
	spec := Spec{
		Scenario: "needle",
		Probs:    []float64{0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.3},
		Policies: []string{"epsgreedy:0.1", "epsgreedy:0.01", "ucb1", "thompson"},
		Steps:    2000,
		Seeds:    40,
		BaseSeed: 1,
	}
	for _, workers := range []int{1, 2, 4, 16} {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			spec.Workers = workers
			for b.Loop() {
				if _, err := Compare(context.Background(), spec); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
