package ex2

import "testing"

func TestParseProbability(t *testing.T) {
	good := map[string]float64{"0": 0, "1": 1, "0.25": 0.25, "1e-1": 0.1}
	for in, want := range good {
		got, err := ParseProbability(in)
		if err != nil || got != want {
			t.Errorf("ParseProbability(%q) = %v, %v; want %v, nil", in, got, err, want)
		}
	}
	for _, in := range []string{"", "half", "1.5", "-0.1", "NaN"} {
		if got, err := ParseProbability(in); err == nil {
			t.Errorf("ParseProbability(%q) = %v, nil; want an error", in, got)
		}
	}
}
