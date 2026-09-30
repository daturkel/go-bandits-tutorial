// Package config reads the daemon's settings from flags and the environment.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"time"
)

// Config holds every setting banditd needs.
type Config struct {
	Addr          string        // listen address
	Policy        string        // policy spec, e.g. thompson or epsgreedy:0.05
	Arms          int           // number of arms
	Seed          uint64        // seed for the policy's random generator
	LogLevel      slog.Level    // minimum level to log
	LogFormat     string        // "text" or "json"
	ShutdownGrace time.Duration // how long in-flight requests get on shutdown
	SelectDelay   time.Duration // artificial latency added to /select
	DrainDelay    time.Duration // keep accepting this long after shutdown starts, failing health checks

	DatabaseURL    string        // PostgreSQL URL; empty means keep state in memory only
	PendingTTL     time.Duration // how long a selection waits for its reward
	ExpireInterval time.Duration // how often expired selections are swept
	ExpireReward   *float64      // reward to count for selections that expire unrewarded; nil means ignore them

	DebugAddr string // address for the profiling server (pprof); empty disables it
}

// ErrInvalid marks a bad setting, wherever it came from.
var ErrInvalid = errors.New("invalid configuration")

// Load builds a Config from command-line arguments and the environment.
//
// Each setting can be given as a flag (-addr) or an environment variable
// (BANDIT_ADDR). The order of precedence is: flag, then environment variable,
// then the built-in default. getenv is a parameter, not a call to os.Getenv,
// so tests can supply a fake environment; usage and flag errors are written
// to stderr.
func Load(args []string, getenv func(string) string, stderr io.Writer) (Config, error) {
	var errs []error
	env := envReader{getenv: getenv, errs: &errs}

	var cfg Config
	fs := flag.NewFlagSet("banditd", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&cfg.Addr, "addr", env.str("BANDIT_ADDR", "localhost:8080"), "listen `address` (env BANDIT_ADDR)")
	fs.StringVar(&cfg.Policy, "policy", env.str("BANDIT_POLICY", "thompson"), "policy `spec`: thompson, ucb1, epsgreedy[:eps] (env BANDIT_POLICY)")
	fs.IntVar(&cfg.Arms, "arms", env.int("BANDIT_ARMS", 3), "number of arms (env BANDIT_ARMS)")
	fs.Uint64Var(&cfg.Seed, "seed", env.uint64("BANDIT_SEED", 1), "policy random seed (env BANDIT_SEED)")
	level := fs.String("log-level", env.str("BANDIT_LOG_LEVEL", "info"), "debug, info, warn or error (env BANDIT_LOG_LEVEL)")
	fs.StringVar(&cfg.LogFormat, "log-format", env.str("BANDIT_LOG_FORMAT", "text"), "text or json (env BANDIT_LOG_FORMAT)")
	fs.DurationVar(&cfg.ShutdownGrace, "shutdown-grace", env.duration("BANDIT_SHUTDOWN_GRACE", 10*time.Second), "time for in-flight requests to finish on shutdown (env BANDIT_SHUTDOWN_GRACE)")
	fs.DurationVar(&cfg.DrainDelay, "drain-delay", env.duration("BANDIT_DRAIN_DELAY", 0), "how long to keep serving, reporting unhealthy, after a shutdown signal (env BANDIT_DRAIN_DELAY)")
	fs.StringVar(&cfg.DatabaseURL, "database-url", env.str("BANDIT_DATABASE_URL", ""), "PostgreSQL `url`; empty keeps state in memory only (env BANDIT_DATABASE_URL)")
	fs.DurationVar(&cfg.PendingTTL, "pending-ttl", env.duration("BANDIT_PENDING_TTL", 10*time.Minute), "how long a selection waits for its reward (env BANDIT_PENDING_TTL)")
	fs.DurationVar(&cfg.ExpireInterval, "expire-interval", env.duration("BANDIT_EXPIRE_INTERVAL", 30*time.Second), "how often expired selections are swept (env BANDIT_EXPIRE_INTERVAL)")
	expireReward := fs.String("expire-reward", env.str("BANDIT_EXPIRE_REWARD", ""), "count unrewarded expired selections as this `reward` (for example 0); empty ignores them (env BANDIT_EXPIRE_REWARD)")
	fs.StringVar(&cfg.DebugAddr, "debug-addr", env.str("BANDIT_DEBUG_ADDR", ""), "serve pprof profiles on this `address`, e.g. localhost:6060; empty disables (env BANDIT_DEBUG_ADDR)")
	fs.DurationVar(&cfg.SelectDelay, "select-delay", env.duration("BANDIT_SELECT_DELAY", 0), "artificial latency for /select (env BANDIT_SELECT_DELAY)")

	if err := fs.Parse(args); err != nil {
		return Config{}, err // includes flag.ErrHelp for -h
	}

	if err := cfg.LogLevel.UnmarshalText([]byte(*level)); err != nil {
		errs = append(errs, fmt.Errorf("log level %q: %w", *level, err))
	}
	if *expireReward != "" {
		r, err := strconv.ParseFloat(*expireReward, 64)
		if err != nil || !(r >= 0 && r <= 1) {
			errs = append(errs, fmt.Errorf("expire reward %q must be a number between 0 and 1", *expireReward))
		} else {
			cfg.ExpireReward = &r
		}
	}
	if cfg.PendingTTL <= 0 || cfg.ExpireInterval <= 0 {
		errs = append(errs, errors.New("pending-ttl and expire-interval must be positive"))
	}
	if cfg.Arms < 1 {
		errs = append(errs, fmt.Errorf("arms must be at least 1, got %d", cfg.Arms))
	}
	if cfg.LogFormat != "text" && cfg.LogFormat != "json" {
		errs = append(errs, fmt.Errorf("log format must be text or json, got %q", cfg.LogFormat))
	}
	if cfg.ShutdownGrace < 0 || cfg.SelectDelay < 0 || cfg.DrainDelay < 0 {
		errs = append(errs, errors.New("durations must not be negative"))
	}
	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	return cfg, nil
}

// envReader reads typed values from the environment. A value that does not
// parse is recorded as an error, naming the variable, and the default is
// returned so that Load can report every problem at once.
type envReader struct {
	getenv func(string) string
	errs   *[]error
}

func (e envReader) str(key, def string) string {
	if v := e.getenv(key); v != "" {
		return v
	}
	return def
}

func (e envReader) int(key string, def int) int {
	v := e.getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		*e.errs = append(*e.errs, fmt.Errorf("%s=%q: %w", key, v, err))
		return def
	}
	return n
}

func (e envReader) uint64(key string, def uint64) uint64 {
	v := e.getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		*e.errs = append(*e.errs, fmt.Errorf("%s=%q: %w", key, v, err))
		return def
	}
	return n
}

func (e envReader) duration(key string, def time.Duration) time.Duration {
	v := e.getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		*e.errs = append(*e.errs, fmt.Errorf("%s=%q: %w", key, v, err))
		return def
	}
	return d
}
