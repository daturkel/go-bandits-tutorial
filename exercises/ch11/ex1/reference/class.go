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
	responses := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_responses_total",
		Help: "HTTP responses by status class.",
	}, []string{"class"})
	reg.MustRegister(responses) // once, when the middleware is built

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &recorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			responses.WithLabelValues(class(rec.status)).Inc()
		})
	}
}

// recorder remembers the status written; 200 unless WriteHeader says otherwise.
type recorder struct {
	http.ResponseWriter
	status int
}

func (r *recorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func class(status int) string {
	switch {
	case status >= 500:
		return "5xx"
	case status >= 400:
		return "4xx"
	case status >= 300:
		return "3xx"
	case status >= 200:
		return "2xx"
	default:
		return "1xx"
	}
}
