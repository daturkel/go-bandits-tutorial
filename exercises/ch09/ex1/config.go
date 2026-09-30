package ex1

import "io"

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
	// TODO
	return Config{}, nil
}
