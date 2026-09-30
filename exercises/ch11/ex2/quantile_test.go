package ex2

import (
	"math"
	"testing"
)

func TestEstimateQuantile(t *testing.T) {
	// 100 observations: 10 in (0, 1], 40 in (1, 2], 40 in (2, 5], 10 in (5, 10].
	upper := []float64{1, 2, 5, 10}
	counts := []uint64{10, 40, 40, 10}

	tests := []struct {
		name string
		q    float64
		want float64
	}{
		{"median lands on the top of the second bucket", 0.5, 1 + (50.0-10)/40}, // rank 50; 10 observations lie below, 40 in the bucket
		{"low quantile in first bucket", 0.05, 0.5},
		{"quantile on a bucket edge", 0.10, 1},
		{"in the third bucket", 0.75, 2 + (75.0-50)/40*3},
		{"in the last bucket", 0.95, 5 + (95.0-90)/10*5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EstimateQuantile(tc.q, upper, counts)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("EstimateQuantile(%v) = %v, want %v", tc.q, got, tc.want)
			}
		})
	}
}

func TestEstimateQuantileEdgeCases(t *testing.T) {
	upper := []float64{1, 2}
	if got := EstimateQuantile(0.5, upper, []uint64{0, 0}); !math.IsNaN(got) {
		t.Errorf("no observations: got %v, want NaN", got)
	}
	for _, q := range []float64{0, 1, -0.5, 2, math.NaN()} {
		if got := EstimateQuantile(q, upper, []uint64{1, 1}); !math.IsNaN(got) {
			t.Errorf("q=%v: got %v, want NaN", q, got)
		}
	}
	if got := EstimateQuantile(0.5, upper, []uint64{1}); !math.IsNaN(got) {
		t.Errorf("mismatched lengths: got %v, want NaN", got)
	}
	// All observations in one bucket: interpolates across that bucket.
	if got := EstimateQuantile(0.5, []float64{10}, []uint64{8}); math.Abs(got-5) > 1e-9 {
		t.Errorf("single bucket median = %v, want 5", got)
	}
	// Empty buckets before the answer are skipped.
	if got := EstimateQuantile(0.5, []float64{1, 2, 3}, []uint64{0, 0, 4}); math.Abs(got-2.5) > 1e-9 {
		t.Errorf("median with empty leading buckets = %v, want 2.5", got)
	}
}
