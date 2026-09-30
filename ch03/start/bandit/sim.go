package bandit

// Result summarises one simulation run.
type Result struct {
	Policy      string
	Steps       int
	TotalReward float64
	Regret      float64   // cumulative regret after the last step
	RegretCurve []float64 // cumulative regret after each step
	Pulls       []int     // how often each arm was pulled
}

// Run plays steps rounds of pol against env. It knows nothing about which
// concrete policy it is driving. A policy that returns an arm the environment
// does not have stops the run with ErrArmOutOfRange; the partial Result so far
// is returned alongside the error.
//
// Regret is measured in expectation: each pull costs the gap between the best
// arm's probability and the pulled arm's probability, so lucky draws do not
// hide bad decisions.
func Run(env *Env, pol Policy, steps int) (Result, error) {
	_, bestP := env.Best()
	res := Result{
		Policy:      pol.Name(),
		Steps:       steps,
		RegretCurve: make([]float64, steps),
		Pulls:       make([]int, env.NumArms()),
	}
	for t := range steps {
		arm := pol.Select()
		// TASK 6: a policy is now something anyone can write, so do not trust
		// the arm it returns. If it is not a valid arm of env, stop and return
		// res so far with an error that wraps ErrArmOutOfRange and says the
		// step, the policy's name, the arm and how many arms there are.
		reward := env.Pull(arm)
		pol.Update(arm, reward)

		res.TotalReward += reward
		res.Regret += bestP - env.Prob(arm)
		res.RegretCurve[t] = res.Regret
		res.Pulls[arm]++
	}
	return res, nil
}
