package ex3

import (
	"math"
	"testing"
)

func sum(p []float64) (s float64) {
	for _, v := range p {
		s += v
	}
	return s
}

func TestSoftmaxUniformWhenScoresAreEqual(t *testing.T) {
	p := Softmax([]float64{3, 3, 3, 3}, 1)
	for i, v := range p {
		if math.Abs(v-0.25) > 1e-12 {
			t.Errorf("p[%d] = %v, want 0.25", i, v)
		}
	}
}

func TestSoftmaxKnownValues(t *testing.T) {
	// exp(0), exp(1) normalised: 1/(1+e) and e/(1+e).
	p := Softmax([]float64{0, 1}, 1)
	e := math.E
	if len(p) != 2 || math.Abs(p[0]-1/(1+e)) > 1e-12 || math.Abs(p[1]-e/(1+e)) > 1e-12 {
		t.Errorf("Softmax([0 1], 1) = %v", p)
	}
	// Temperature 0.5 doubles the scores.
	q := Softmax([]float64{0, 1}, 0.5)
	if math.Abs(q[1]-math.E*math.E/(1+math.E*math.E)) > 1e-12 {
		t.Errorf("Softmax([0 1], 0.5) = %v", q)
	}
}

func TestSoftmaxSurvivesLargeScores(t *testing.T) {
	p := Softmax([]float64{5000, 5001, 4990}, 1)
	if len(p) != 3 || math.IsNaN(sum(p)) || math.Abs(sum(p)-1) > 1e-12 {
		t.Fatalf("Softmax of large scores = %v (sum %v)", p, sum(p))
	}
	if !(p[1] > p[0] && p[0] > p[2]) {
		t.Errorf("order is wrong: %v", p)
	}
	tiny := Softmax([]float64{-5000, -5001}, 1)
	if math.Abs(sum(tiny)-1) > 1e-12 {
		t.Errorf("Softmax of very negative scores sums to %v", sum(tiny))
	}
}

func TestSoftmaxLowTemperatureIsNearlyGreedy(t *testing.T) {
	p := Softmax([]float64{1, 2, 1.5}, 0.01)
	if p[1] < 0.999 {
		t.Errorf("p = %v, want nearly all mass on arm 1", p)
	}
}

func TestSoftmaxBadInput(t *testing.T) {
	if Softmax(nil, 1) != nil || Softmax([]float64{1}, 0) != nil || Softmax([]float64{1}, -1) != nil {
		t.Error("empty scores or a non-positive temperature should give nil")
	}
}
