package bandit

import (
	"maps"
	"slices"
)

// scenarios maps a name to the per-arm success probabilities of a test bed.
var scenarios = map[string][]float64{
	"easy":   {0.2, 0.5, 0.8},
	"close":  {0.50, 0.55, 0.60},
	"needle": {0.10, 0.10, 0.10, 0.10, 0.10, 0.10, 0.10, 0.10, 0.10, 0.30},
	"spread": {0.05, 0.15, 0.25, 0.35, 0.45, 0.55, 0.65, 0.75},
}

// Scenario looks up a named set of arm probabilities.
func Scenario(name string) ([]float64, error) {
	// TASK 3: an unknown name must be an error wrapping ErrUnknownScenario
	// whose message names the culprit and lists ScenarioNames(). A known name
	// returns a copy of its probabilities. You will need "fmt" and "strings".
	probs := scenarios[name]
	return slices.Clone(probs), nil
}

// ScenarioNames lists the known scenarios in alphabetical order.
func ScenarioNames() []string {
	return slices.Sorted(maps.Keys(scenarios))
}
