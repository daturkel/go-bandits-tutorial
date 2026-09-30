// Package daemon is what the three gRPC programs share: reading settings from
// flags and the environment, building the logger, and turning signals into a
// cancelled context.
package daemon

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"banditlab/internal/logging"
)

// ErrInvalid marks a bad setting, wherever it came from.
var ErrInvalid = errors.New("invalid configuration")

// Common is the settings every daemon has.
type Common struct {
	Addr      string
	LogLevel  slog.Level
	LogFormat string
	Grace     time.Duration // how long calls in progress get to finish on shutdown
	Deadline  time.Duration // applied to unary calls that arrive without one
}

// Env reads typed values from the environment, remembering every value that
// did not parse so that Err can report them all at once.
type Env struct {
	getenv func(string) string
	errs   []error
}

// NewEnv returns an Env reading through getenv (os.Getenv, or a fake in tests).
func NewEnv(getenv func(string) string) *Env { return &Env{getenv: getenv} }

// Str returns the variable's value, or def if it is unset or empty.
func (e *Env) Str(key, def string) string {
	if v := e.getenv(key); v != "" {
		return v
	}
	return def
}

// Int is Str for integers.
func (e *Env) Int(key string, def int) int {
	v := e.getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s=%q: %w", key, v, err))
		return def
	}
	return n
}

// Uint64 is Str for unsigned integers.
func (e *Env) Uint64(key string, def uint64) uint64 {
	v := e.getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s=%q: %w", key, v, err))
		return def
	}
	return n
}

// Duration is Str for durations such as "30s".
func (e *Env) Duration(key string, def time.Duration) time.Duration {
	v := e.getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s=%q: %w", key, v, err))
		return def
	}
	return d
}

// Err returns the problems found so far, wrapped in ErrInvalid, or nil.
func (e *Env) Err() error {
	if len(e.errs) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrInvalid, errors.Join(e.errs...))
}

// Register adds the common flags to fs. Each can also be set through the
// environment (BANDIT_ADDR, BANDIT_LOG_LEVEL, ...); a flag wins over the
// environment, which wins over the default. Call Finish after fs.Parse.
func (c *Common) Register(fs *flag.FlagSet, env *Env, defaultAddr string) (finish func() error) {
	fs.StringVar(&c.Addr, "addr", env.Str("BANDIT_ADDR", defaultAddr), "listen `address` (env BANDIT_ADDR)")
	level := fs.String("log-level", env.Str("BANDIT_LOG_LEVEL", "info"), "debug, info, warn or error (env BANDIT_LOG_LEVEL)")
	fs.StringVar(&c.LogFormat, "log-format", env.Str("BANDIT_LOG_FORMAT", "text"), "text or json (env BANDIT_LOG_FORMAT)")
	fs.DurationVar(&c.Grace, "shutdown-grace", env.Duration("BANDIT_SHUTDOWN_GRACE", 10*time.Second), "time for calls in progress to finish on shutdown (env BANDIT_SHUTDOWN_GRACE)")
	fs.DurationVar(&c.Deadline, "default-deadline", env.Duration("BANDIT_DEFAULT_DEADLINE", 5*time.Second), "deadline for calls that arrive without one (env BANDIT_DEFAULT_DEADLINE)")
	return func() error {
		var errs []error
		if err := c.LogLevel.UnmarshalText([]byte(*level)); err != nil {
			errs = append(errs, fmt.Errorf("log level %q: %w", *level, err))
		}
		if c.LogFormat != "text" && c.LogFormat != "json" {
			errs = append(errs, fmt.Errorf("log format must be text or json, got %q", c.LogFormat))
		}
		if c.Grace < 0 || c.Deadline <= 0 {
			errs = append(errs, errors.New("shutdown-grace must not be negative and default-deadline must be positive"))
		}
		if err := errors.Join(errs...); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalid, err)
		}
		return nil
	}
}

// Logger builds the daemon's logger, tagged with its name.
func (c *Common) Logger(w io.Writer, name string) (*slog.Logger, error) {
	log, err := logging.New(w, c.LogFormat, c.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	return log.With("service", name), nil
}

// Main is the whole of a daemon's main function: cancel the context on
// SIGINT or SIGTERM, run, and turn the result into an exit code (0 for a
// clean stop or -h, 2 for a bad setting, 1 for anything else).
func Main(name string, run func(ctx context.Context, args []string, getenv func(string) string, stderr io.Writer) error) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		stop() // a second signal kills us immediately
	}()
	err := run(ctx, os.Args[1:], os.Getenv, os.Stderr)
	stop()

	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
	case errors.Is(err, ErrInvalid):
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
		os.Exit(2)
	default:
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
		os.Exit(1)
	}
}
