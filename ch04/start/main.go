package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "banditsim:", err)
		if errors.Is(err, errUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

// errUsage marks mistakes in how the program was invoked.
var errUsage = errors.New("usage error")

const usage = "usage: banditsim run|compare [flags]"

// run is main without the process-level side effects, so tests can call it.
func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", errUsage, usage)
	}
	switch cmd, rest := args[0], args[1:]; cmd {
	case "run":
		return runCmd(rest, out)
	case "compare":
		return compareCmd(rest, out)
	default:
		return fmt.Errorf("%w: unknown command %q; %s", errUsage, cmd, usage)
	}
}
