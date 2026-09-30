// Package metrics defines what the service reports to Prometheus.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"banditlab/internal/bandit"
)

// Metrics holds every metric the service exports, on its own registry (so
// tests can create as many as they like without clashing over global state).
type Metrics struct {
	registry *prometheus.Registry

	requests   *prometheus.CounterVec
	duration   *prometheus.HistogramVec
	inFlight   prometheus.Gauge
	selections *prometheus.CounterVec
	rewards    *prometheus.CounterVec
	expired    prometheus.Counter
	storeErrs  prometheus.Counter
}

// New creates the metrics and their registry.
func New() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "banditd_http_requests_total",
			Help: "HTTP requests served, by method, route pattern and status code.",
		}, []string{"method", "route", "code"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "banditd_http_request_duration_seconds",
			Help: "Time from receiving a request to finishing its response, by route pattern.",
			// Handlers here take microseconds to milliseconds; the default
			// buckets start at 5 ms and would put everything in the first one.
			Buckets: prometheus.ExponentialBuckets(0.00005, 2, 16), // 50 µs ... ~1.6 s
		}, []string{"route"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "banditd_http_requests_in_flight",
			Help: "Requests currently being handled.",
		}),
		selections: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "banditd_selections_total",
			Help: "Arms handed out, by arm.",
		}, []string{"arm"}),
		rewards: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "banditd_reward_total",
			Help: "Sum of rewards recorded, by arm.",
		}, []string{"arm"}),
		expired: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "banditd_selections_expired_total",
			Help: "Selections that expired without a reward.",
		}),
		storeErrs: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "banditd_store_errors_total",
			Help: "Failed calls to the store.",
		}),
	}
	m.registry.MustRegister(
		m.requests, m.duration, m.inFlight, m.selections, m.rewards, m.expired, m.storeErrs,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return m
}

// TrackArms exports per-arm gauges computed from snapshot at every scrape. Call
// it at most once per Metrics.
func (m *Metrics) TrackArms(snapshot func() []bandit.ArmStat) {
	m.registry.MustRegister(&armCollector{snapshot: snapshot})
}

// Handler serves the metrics in Prometheus text format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Selected records that arm was handed out.
func (m *Metrics) Selected(arm int) { m.selections.WithLabelValues(strconv.Itoa(arm)).Inc() }

// Rewarded records a reward for arm.
func (m *Metrics) Rewarded(arm int, reward float64) {
	m.rewards.WithLabelValues(strconv.Itoa(arm)).Add(reward)
}

// Expired records n selections that expired unrewarded.
func (m *Metrics) Expired(n int) { m.expired.Add(float64(n)) }

// StoreError records a failed store call.
func (m *Metrics) StoreError() { m.storeErrs.Inc() }

// Middleware records a count and a duration for every request.
//
// The route label is the ServeMux pattern that matched ("POST /select"), not
// the URL path. A label's every distinct value creates a new time series, so
// a label built from user input (a raw path, an id) would grow without bound.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m.inFlight.Inc()
		defer m.inFlight.Dec()

		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		// The mux fills in r.Pattern while routing, so it is only known now.
		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		code := rec.status
		if code == 0 {
			code = http.StatusOK
		}
		m.requests.WithLabelValues(r.Method, route, strconv.Itoa(code)).Inc()
		m.duration.WithLabelValues(route).Observe(time.Since(start).Seconds())
	})
}

// statusRecorder remembers the status code the handler wrote.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// armCollector reports per-arm state as gauges. Unlike counters, which the
// code increments as events happen, these are computed from the policy each
// time Prometheus scrapes, by implementing prometheus.Collector.
type armCollector struct {
	snapshot func() []bandit.ArmStat
}

var (
	pullsDesc = prometheus.NewDesc("banditd_arm_pulls", "Rewards the policy has learned from, by arm.", []string{"arm"}, nil)
	meanDesc  = prometheus.NewDesc("banditd_arm_mean", "The policy's current estimate of an arm's success rate.", []string{"arm"}, nil)
)

func (c *armCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- pullsDesc
	ch <- meanDesc
}

func (c *armCollector) Collect(ch chan<- prometheus.Metric) {
	for _, a := range c.snapshot() {
		arm := strconv.Itoa(a.Arm)
		ch <- prometheus.MustNewConstMetric(pullsDesc, prometheus.GaugeValue, float64(a.Pulls), arm)
		ch <- prometheus.MustNewConstMetric(meanDesc, prometheus.GaugeValue, a.Mean, arm)
	}
}
