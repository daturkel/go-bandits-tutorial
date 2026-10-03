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
	// TASK 8: for each step, ask pol.Select() for an arm, pull it in env,
	// tell the policy with pol.Update(arm, reward), and record everything in
	// the Result: TotalReward, Regret (add best - env.Prob(arm)),
	// RegretCurve[t], and Pulls[arm]. Return the Result.
	return Result{}
}
