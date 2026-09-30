package bandit

import (
	"math"
	"testing"
)

func TestSampleBetaMoments(t *testing.T) {
	tests := []struct{ a, b float64 }{{1, 1}, {2, 5}, {30, 10}, {0.5, 0.5}, {1, 100}}
	for _, tc := range tests {
		rng := NewRNG(99, StreamPolicy)
		const n = 100_000
		var sum, sumSq float64
		for range n {
			x := SampleBeta(tc.a, tc.b, rng)
			if x < 0 || x > 1 {
				t.Fatalf("Beta(%v,%v) sample %v outside [0,1]", tc.a, tc.b, x)
			}
			sum += x
			sumSq += x * x
		}
		mean := sum / n
		variance := sumSq/n - mean*mean
		wantMean := tc.a / (tc.a + tc.b)
		wantVar := tc.a * tc.b / ((tc.a + tc.b) * (tc.a + tc.b) * (tc.a + tc.b + 1))
		if math.Abs(mean-wantMean) > 0.005 {
			t.Errorf("Beta(%v,%v) mean = %.4f, want %.4f", tc.a, tc.b, mean, wantMean)
		}
		if math.Abs(variance-wantVar) > 0.003 {
			t.Errorf("Beta(%v,%v) variance = %.4f, want %.4f", tc.a, tc.b, variance, wantVar)
		}
	}
}
