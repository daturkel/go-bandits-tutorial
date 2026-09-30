package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
)

func main() {
	// The context is cancelled by Ctrl-C (SIGINT). Everything below watches it.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	go func() {
		<-ctx.Done()
		stop() // restore default handling, so a second Ctrl-C kills us immediately
	}()

	err := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "banditsim:", err)
		if errors.Is(err, errUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

// errUsage marks mistakes in how the program was invoked.
var errUsage = errors.New("usage error")

const usage = "usage: banditsim run|compare|serve [flags]"

// run is main without the process-level side effects, so tests can call it.
// Results go to stdout; progress and diagnostics go to stderr.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", errUsage, usage)
	}
	switch cmd, rest := args[0], args[1:]; cmd {
	case "run":
		return runCmd(ctx, rest, stdout)
	case "compare":
		return compareCmd(ctx, rest, stdout, stderr)
	case "serve":
		return serveCmd(ctx, rest, stderr)
	default:
		return fmt.Errorf("%w: unknown command %q; %s", errUsage, cmd, usage)
	}
}
