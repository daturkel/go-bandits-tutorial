package ex1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// counts gathers the registry and returns the counter values by class label.
func counts(t *testing.T, reg *prometheus.Registry) map[string]float64 {
	t.Helper()
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]float64{}
	for _, f := range families {
		if f.GetName() != "http_responses_total" {
			continue
		}
		if f.GetType() != dto.MetricType_COUNTER {
			t.Fatalf("http_responses_total is a %v, want a counter", f.GetType())
		}
		for _, m := range f.GetMetric() {
			if len(m.GetLabel()) != 1 || m.GetLabel()[0].GetName() != "class" {
				t.Fatalf("labels = %v, want exactly one label named class", m.GetLabel())
			}
			out[m.GetLabel()[0].GetValue()] = m.GetCounter().GetValue()
		}
	}
	return out
}

func TestCountsByStatusClass(t *testing.T) {
	reg := prometheus.NewRegistry()
	h := CountByClass(reg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/created":
			w.WriteHeader(http.StatusCreated)
		case "/missing":
			http.NotFound(w, r)
		case "/teapot":
			w.WriteHeader(http.StatusTeapot)
		case "/boom":
			w.WriteHeader(http.StatusInternalServerError)
		case "/implicit":
			w.Write([]byte("ok")) // no WriteHeader: 200
		case "/silent":
			// writes nothing at all: also 200
		}
	}))
	for _, path := range []string{"/created", "/created", "/missing", "/teapot", "/boom", "/implicit", "/silent"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", path, nil))
	}
	got := counts(t, reg)
	want := map[string]float64{"2xx": 4, "4xx": 2, "5xx": 1}
	for class, n := range want {
		if got[class] != n {
			t.Errorf("class %s = %v, want %v (all: %v)", class, got[class], n, got)
		}
	}
	if len(got) != len(want) {
		t.Errorf("unexpected classes reported: %v", got)
	}
}

// The counter must not be created per request: two middlewares on the same
// registry would collide, but one middleware serving many requests must not.
func TestManyRequestsShareOneCounter(t *testing.T) {
	reg := prometheus.NewRegistry()
	h := CountByClass(reg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	for range 100 {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	}
	if got := counts(t, reg)["2xx"]; got != 100 {
		t.Errorf("2xx = %v, want 100", got)
	}
}
