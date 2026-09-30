// Command feedbackd is one of the gRPC services; see its package for what it does.
package main

import (
	"banditlab/internal/apps/feedbackd"
	"banditlab/internal/daemon"
)

func main() { daemon.Main("feedbackd", feedbackd.Run) }
