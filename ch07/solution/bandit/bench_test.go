package bandit

import (
	"context"
	"testing"
)

func BenchmarkSelectUpdate(b *testing.B) {
	for _, spec := range PolicyNames() {
		b.Run(spec, func(b *testing.B) {
			pol, err := NewPolicy(spec, 10, NewRNG(1, StreamPolicy))
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				arm := pol.Select()
				pol.Update(arm, 1)
			}
		})
	}
}

func BenchmarkRun(b *testing.B) {
	probs, _ := Scenario("spread")
	for b.Loop() {
		env, _ := NewEnv(probs, NewRNG(1, StreamEnv))
		pol, _ := NewPolicy("ucb1", env.NumArms(), NewRNG(1, StreamPolicy))
		if _, err := Run(context.Background(), env, pol, 10_000); err != nil {
			b.Fatal(err)
		}
	}
}
