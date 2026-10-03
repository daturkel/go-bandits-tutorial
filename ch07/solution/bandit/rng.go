package bandit

import "math/rand/v2"

// Every source of randomness gets its own stream, so changing how many
// numbers the policy draws never changes what the environment does.
const (
	StreamEnv uint64 = iota + 1
	StreamPolicy
)

// NewRNG returns a deterministic generator: same seed and stream, same numbers.
func NewRNG(seed, stream uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, stream))
}
