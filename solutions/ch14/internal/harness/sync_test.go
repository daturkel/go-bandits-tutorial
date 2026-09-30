package harness

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func syncSpec() SyncSpec {
	return SyncSpec{
		Scenario: "easy", Probs: []float64{0.2, 0.5, 0.8},
		Policies: []string{"thompson", "ucb1"}, Replicas: 4, Steps: 2000, Seeds: 10, BaseSeed: 1,
		Intervals: []int{0, 100, 1},
	}
}

func TestSweepSyncMoreSyncingMeansLessRegret(t *testing.T) {
	points, err := SweepSync(context.Background(), syncSpec())
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 6 {
		t.Fatalf("got %d points, want 6 (2 policies x 3 intervals)", len(points))
	}
	for p := range 2 {
		never, often := points[p*3], points[p*3+2]
		if never.Interval != 0 || often.Interval != 1 {
			t.Fatalf("points are not in (policy, interval) order: %+v", points)
		}
		if often.Regret >= never.Regret {
			t.Errorf("%s: syncing every step (%.1f) should beat never syncing (%.1f)", never.Policy, often.Regret, never.Regret)
		}
	}
}

func TestSweepSyncIsDeterministic(t *testing.T) {
	a, _ := SweepSync(context.Background(), syncSpec())
	b, _ := SweepSync(context.Background(), syncSpec())
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("point %d differs between runs: %+v vs %+v", i, a[i], b[i])
		}
	}
}

func TestSweepSyncRejectsBadInput(t *testing.T) {
	for name, mutate := range map[string]func(*SyncSpec){
		"no replicas":       func(s *SyncSpec) { s.Replicas = 0 },
		"negative interval": func(s *SyncSpec) { s.Intervals = []int{-5} },
		"no policies":       func(s *SyncSpec) { s.Policies = nil },
		"unknown policy":    func(s *SyncSpec) { s.Policies = []string{"nope"} },
	} {
		t.Run(name, func(t *testing.T) {
			spec := syncSpec()
			mutate(&spec)
			if _, err := SweepSync(context.Background(), spec); err == nil {
				t.Error("want an error")
			}
		})
	}
}

func TestSyncOutputs(t *testing.T) {
	spec := syncSpec()
	points, _ := SweepSync(context.Background(), spec)
	var svg, csv bytes.Buffer
	if err := WriteSyncSVG(&svg, spec, points); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(svg.String(), "never") || strings.Count(svg.String(), "<polyline") != 2 {
		t.Errorf("unexpected SVG:\n%s", svg.String())
	}
	if err := WriteSyncCSV(&csv, points); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(csv.String(), "\n"); got != 7 {
		t.Errorf("CSV has %d lines, want 7", got)
	}
}

func TestCompareWithDelay(t *testing.T) {
	spec := testSpec()
	spec.Policies = []string{"ucb1"}
	spec.Delay = 100
	delayed, err := Compare(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	spec.Delay = 0
	immediate, _ := Compare(context.Background(), spec)
	if delayed.Curves[0].Mean[499] <= immediate.Curves[0].Mean[499] {
		t.Errorf("delayed feedback should cost regret: %v vs %v", delayed.Curves[0].Mean[499], immediate.Curves[0].Mean[499])
	}
	spec.Delay = -1
	if _, err := Compare(context.Background(), spec); err == nil {
		t.Error("negative delay should be rejected")
	}
}
