package bandit

import "testing"

func draw(seed, stream uint64) [5]uint64 {
	rng := NewRNG(seed, stream)
	var out [5]uint64
	for i := range out {
		out[i] = rng.Uint64()
	}
	return out
}

func TestNewRNGSameSeedAndStreamGiveTheSameNumbers(t *testing.T) {
	if draw(7, StreamEnv) != draw(7, StreamEnv) {
		t.Error("two generators with the same seed and stream disagree")
	}
}

func TestNewRNGDependsOnTheSeed(t *testing.T) {
	if draw(1, StreamEnv) == draw(2, StreamEnv) {
		t.Error("different seeds gave the same numbers")
	}
}

func TestNewRNGDependsOnTheStream(t *testing.T) {
	if draw(1, StreamEnv) == draw(1, StreamPolicy) {
		t.Error("different streams gave the same numbers: the policy and the environment would move together")
	}
}

func TestStreamsAreDistinctNamedNumbers(t *testing.T) {
	if StreamEnv == StreamPolicy || StreamEnv == 0 || StreamPolicy == 0 {
		t.Errorf("StreamEnv=%d StreamPolicy=%d", StreamEnv, StreamPolicy)
	}
}
