package ex1

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

// CountByClass returns middleware that counts responses in a counter named
// "http_responses_total" with a single label, "class", whose value is the
// status class: "1xx", "2xx", "3xx", "4xx" or "5xx". The counter must be
// registered on reg.
//
// A handler that never calls WriteHeader or Write responds 200, so count that
// as "2xx". Use only the status class as a label: a label per exact status,
// or per URL path, would create a new time series for every distinct value.
func CountByClass(reg prometheus.Registerer) func(http.Handler) http.Handler {
	// TODO: create and register the counter here, once.
	return func(next http.Handler) http.Handler {
		return next
	}
}
