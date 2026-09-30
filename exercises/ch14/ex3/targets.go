package ex3

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
	// TODO
	return nil, nil
}
