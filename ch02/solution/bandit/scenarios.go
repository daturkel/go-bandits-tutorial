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
func Scenario(name string) ([]float64, bool) {
	probs, ok := scenarios[name]
	return slices.Clone(probs), ok
}

// ScenarioNames lists the known scenarios in alphabetical order.
func ScenarioNames() []string {
	return slices.Sorted(maps.Keys(scenarios))
}
