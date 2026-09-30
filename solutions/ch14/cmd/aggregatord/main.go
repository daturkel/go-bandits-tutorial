// Command aggregatord merges counts and streams them to the policy services.
package main

import (
	"banditlab/internal/apps/aggregatord"
	"banditlab/internal/daemon"
)

func main() { daemon.Main("aggregatord", aggregatord.Run) }
