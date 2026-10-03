package harness

import (
	"bytes"
	"encoding/csv"
	"math"
	"strings"
	"testing"
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
