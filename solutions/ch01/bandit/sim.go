package bandit

// Result summarises one simulation run.
type Result struct {
	Steps       int
	TotalReward float64
	Regret      float64   // cumulative regret after the last step
	RegretCurve []float64 // cumulative regret after each step
	Pulls       []int     // how often each arm was pulled
}

// Run plays steps rounds of pol against env.
//
// Regret is measured in expectation: each pull costs the gap between the best
// arm's probability and the pulled arm's probability, so lucky draws do not
// hide bad decisions.
func Run(env *Env, pol *EpsilonGreedy, steps int) Result {
	_, bestP := env.Best()
	res := Result{
		Steps:       steps,
		RegretCurve: make([]float64, steps),
		Pulls:       make([]int, env.NumArms()),
	}
	for t := range steps {
		arm := pol.Select()
		reward := env.Pull(arm)
		pol.Update(arm, reward)

		res.TotalReward += reward
		res.Regret += bestP - env.Prob(arm)
		res.RegretCurve[t] = res.Regret
		res.Pulls[arm]++
	}
	return res
}
