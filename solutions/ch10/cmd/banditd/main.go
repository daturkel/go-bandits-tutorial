// Command banditd serves a bandit policy over HTTP.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"banditlab/internal/config"
)

func main() {
	// SIGTERM is what container runtimes and process managers send to ask for
	// a clean stop; Ctrl-C sends SIGINT.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		stop() // restore default handling, so a second signal kills us immediately
	}()
	err := run(ctx, os.Args[1:], os.Getenv, os.Stderr)
	stop()

	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		return
	case errors.Is(err, config.ErrInvalid):
		fmt.Fprintln(os.Stderr, "banditd:", err)
		os.Exit(2)
	default:
		fmt.Fprintln(os.Stderr, "banditd:", err)
		os.Exit(1)
	}
}
