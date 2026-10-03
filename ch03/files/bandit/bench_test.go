package bandit

import "testing"

// TASK 9: write a benchmark named BenchmarkSelectUpdate.
//
// For each name in PolicyNames(), run a sub-benchmark (b.Run) that builds the
// policy with NewPolicy(spec, 10, NewRNG(1, StreamPolicy)), calls
// b.ReportAllocs(), then loops with `for b.Loop()` doing one Select followed by
// one Update(arm, 1).
func BenchmarkSelectUpdate(b *testing.B) {
	b.Skip("TASK 9: write this benchmark")
}

func BenchmarkRun(b *testing.B) {
	probs, _ := Scenario("spread")
	for b.Loop() {
		env, _ := NewEnv(probs, NewRNG(1, StreamEnv))
		pol, _ := NewPolicy("ucb1", env.NumArms(), NewRNG(1, StreamPolicy))
		if _, err := Run(env, pol, 10_000); err != nil {
			b.Fatal(err)
		}
	}
}
