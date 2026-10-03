package bandit

import "testing"

// TASK 8: write a table-driven test for Scenario.
//
// Each case names a scenario and says what to expect: the probabilities for a
// known name, or an error wrapping ErrUnknownScenario for an unknown one
// (include the empty string and a wrong-case name such as "EASY"). Run each
// case as a subtest with t.Run. For errors, also check the message lists every
// name in ScenarioNames().
//
// Then add a second test that changes the slice one call returns and checks a
// later call is unaffected.
func TestScenario(t *testing.T) {
	t.Skip("TASK 8: write this test")
}
