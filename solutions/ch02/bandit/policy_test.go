package bandit

// Compile-time checks: these lines stop compiling if a type stops satisfying Policy.
var (
	_ Policy = (*EpsilonGreedy)(nil)
	_ Policy = (*UCB1)(nil)
)
