package ex1

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
)

// Config is a tiny service configuration.
type Config struct {
	Port  int
	Debug bool
}

// Load builds a Config from args and the environment.
//
// Settings, with their flag, environment variable and default:
//
//	Port    -port  PORT   8080
//	Debug   -debug DEBUG  false
//
// Precedence: a flag beats the environment variable, which beats the default.
//
// Errors:
//   - An environment value that does not parse (PORT=abc, DEBUG=maybe) is an
//     error whose message contains the variable's name.
//   - A port outside 1-65535 is an error, from any source.
//   - An unknown flag is an error.
//
// Use a flag.FlagSet with flag.ContinueOnError and set its output to stderr,
// so bad input returns an error instead of exiting the program. getenv is
// passed in (rather than calling os.Getenv) so tests can fake the environment.
func Load(args []string, getenv func(string) string, stderr io.Writer) (Config, error) {
	// The environment supplies the defaults the flags start from, so a flag
	// that is present simply overwrites the value.
	cfg := Config{Port: 8080}
	var errs []error
	if v := getenv("PORT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("PORT=%q: %w", v, err))
		} else {
			cfg.Port = n
		}
	}
	if v := getenv("DEBUG"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("DEBUG=%q: %w", v, err))
		} else {
			cfg.Debug = b
		}
	}

	fs := flag.NewFlagSet("service", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.IntVar(&cfg.Port, "port", cfg.Port, "listen port (env PORT)")
	fs.BoolVar(&cfg.Debug, "debug", cfg.Debug, "debug logging (env DEBUG)")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		errs = append(errs, fmt.Errorf("port %d is outside 1-65535", cfg.Port))
	}
	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
