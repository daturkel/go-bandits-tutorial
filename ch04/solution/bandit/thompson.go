package bandit

import "math/rand/v2"

// Thompson implements Thompson sampling for Bernoulli rewards. Each arm keeps
// a Beta(alpha, beta) posterior over its success probability; Select samples
// once from every posterior and plays the arm whose sample is largest.
type Thompson struct {
	alpha, beta []float64 // 1 + successes, 1 + failures (a uniform prior)
	samples     []float64 // scratch space reused by every Select
	rng         *rand.Rand
}

// NewThompson returns a Thompson-sampling policy for nArms arms.
func NewThompson(nArms int, rng *rand.Rand) (*Thompson, error) {
	if nArms < 1 {
		return nil, ErrNoArms
	}
	p := &Thompson{
		alpha:   make([]float64, nArms),
		beta:    make([]float64, nArms),
		samples: make([]float64, nArms),
		rng:     rng,
	}
	for i := range nArms {
		p.alpha[i], p.beta[i] = 1, 1
	}
	return p, nil
}

// Name identifies the policy in reports.
func (p *Thompson) Name() string { return "thompson" }

// Select samples each arm's posterior and returns the arm with the best draw.
func (p *Thompson) Select() int {
	for arm := range p.samples {
		p.samples[arm] = SampleBeta(p.alpha[arm], p.beta[arm], p.rng)
	}
	return Argmax(p.samples, p.rng)
}

// Update adds the reward to the arm's successes and its complement to the failures.
func (p *Thompson) Update(arm int, reward float64) {
	p.alpha[arm] += reward
	p.beta[arm] += 1 - reward
}
