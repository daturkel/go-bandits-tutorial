package harness

import (
	"bytes"
	"encoding/csv"
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
	rep, err := Compare(testSpec())
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
	a, err := Compare(testSpec())
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Compare(testSpec())
	for i := range a.Curves {
		if a.Curves[i].Mean[499] != b.Curves[i].Mean[499] {
			t.Errorf("%s: repeat run differs", a.Curves[i].Policy)
		}
	}
}

func TestCompareRejectsBadInput(t *testing.T) {
	bad := testSpec()
	bad.Policies = []string{"nope"}
	if _, err := Compare(bad); err == nil {
		t.Error("unknown policy should fail")
	}
	bad = testSpec()
	bad.Seeds = 0
	if _, err := Compare(bad); err == nil {
		t.Error("zero seeds should fail")
	}
}

func TestWriteCSV(t *testing.T) {
	rep, _ := Compare(testSpec())
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
	rep, _ := Compare(testSpec())
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

func TestCompareMatchesSequential(t *testing.T) {
	par, err := Compare(testSpec())
	if err != nil {
		t.Fatal(err)
	}
	seq, err := CompareSequential(testSpec())
	if err != nil {
		t.Fatal(err)
	}
	for i := range seq.Curves {
		if par.Curves[i].Policy != seq.Curves[i].Policy {
			t.Fatalf("curve %d: policy %q vs %q", i, par.Curves[i].Policy, seq.Curves[i].Policy)
		}
		for step := range seq.Curves[i].Mean {
			if par.Curves[i].Mean[step] != seq.Curves[i].Mean[step] {
				t.Fatalf("%s: mean differs at step %d", seq.Curves[i].Policy, step)
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
		_, err := Compare(spec)
		if err == nil || !strings.Contains(err.Error(), "first-bad") {
			t.Fatalf("error = %v, want one naming first-bad", err)
		}
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
	b.Run("sequential", func(b *testing.B) {
		for b.Loop() {
			if _, err := CompareSequential(spec); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("goroutines", func(b *testing.B) {
		for b.Loop() {
			if _, err := Compare(spec); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Compare must really run jobs at the same time: with two or more CPUs, a
// comparison big enough to measure takes clearly less wall-clock time than the
// sequential version. The margin is wide (on four cores the goroutine version
// is about three times faster), and each version gets its best of three runs.
func TestCompareRunsInParallel(t *testing.T) {
	if runtime.GOMAXPROCS(0) < 2 {
		t.Skip("needs at least two CPUs")
	}
	spec := testSpec()
	spec.Steps, spec.Seeds = 2000, 48
	best := func(compare func(Spec) (*Report, error)) time.Duration {
		fastest := time.Duration(math.MaxInt64)
		for range 3 {
			start := time.Now()
			if _, err := compare(spec); err != nil {
				t.Fatal(err)
			}
			fastest = min(fastest, time.Since(start))
		}
		return fastest
	}
	seq, par := best(CompareSequential), best(Compare)
	if par > seq*3/4 {
		t.Errorf("Compare took %v and CompareSequential %v: the runs do not seem to overlap", par, seq)
	}
}
