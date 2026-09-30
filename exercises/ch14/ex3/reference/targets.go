package ex3

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ParseTargets splits a comma-separated list of host:port addresses, as given
// in an environment variable such as BANDIT_POLICY_TARGETS, into a slice.
//
//   - Spaces around each entry are ignored, and empty entries (a trailing
//     comma) are skipped.
//   - Every entry must have a host and a port, and the port must be a number
//     from 1 to 65535. IPv6 addresses are written [::1]:9090.
//   - Duplicate entries are an error: they would make a load balancer send
//     twice the traffic to one replica.
//   - An empty list is an error.
//
// The error should say which entry is at fault.
func ParseTargets(spec string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	for entry := range strings.SplitSeq(spec, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		host, port, err := net.SplitHostPort(entry)
		if err != nil || host == "" {
			return nil, fmt.Errorf("target %q: want host:port", entry)
		}
		if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
			return nil, fmt.Errorf("target %q: port must be a number from 1 to 65535", entry)
		}
		if seen[entry] {
			return nil, fmt.Errorf("target %q is listed twice", entry)
		}
		seen[entry] = true
		out = append(out, entry)
	}
	if len(out) == 0 {
		return nil, errors.New("no targets given")
	}
	return out, nil
}
