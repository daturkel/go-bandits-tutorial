package bandit

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestScenario(t *testing.T) {
	tests := []struct {
		name      string
		wantProbs []float64 // nil when an error is expected
		wantErr   error
	}{
		{name: "easy", wantProbs: []float64{0.2, 0.5, 0.8}},
		{name: "close", wantProbs: []float64{0.50, 0.55, 0.60}},
		{name: "nope", wantErr: ErrUnknownScenario},
		{name: "", wantErr: ErrUnknownScenario},
		{name: "EASY", wantErr: ErrUnknownScenario}, // names are case-sensitive
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			probs, err := Scenario(tc.name)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want one wrapping %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				// The message should help: it lists what would have worked.
				for _, known := range ScenarioNames() {
					if !strings.Contains(err.Error(), known) {
						t.Errorf("error %q does not list the known scenario %q", err, known)
					}
				}
				return
			}
			if !slices.Equal(probs, tc.wantProbs) {
				t.Errorf("probs = %v, want %v", probs, tc.wantProbs)
			}
		})
	}
}

func TestScenarioReturnsACopy(t *testing.T) {
	a, _ := Scenario("easy")
	a[0] = 0.99
	b, _ := Scenario("easy")
	if b[0] != 0.2 {
		t.Errorf("changing one caller's slice changed the scenario: %v", b)
	}
}
