package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"banditlab/internal/bandit"
)

func newPolicy(t *testing.T) bandit.SnapshotPolicy {
	t.Helper()
	pol, err := bandit.NewSnapshotPolicy("ucb1", 3, bandit.NewRNG(1, bandit.StreamPolicy))
	if err != nil {
		t.Fatal(err)
	}
	return pol
}

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// do sends a request straight to the handler, without a network.
func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("response %q is not the expected JSON: %v", rec.Body.String(), err)
	}
	return v
}

func TestHealthz(t *testing.T) {
	rec := do(New(newPolicy(t), quietLogger()).Handler(), "GET", "/healthz", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := decode[map[string]string](t, rec)["status"]; got != "ok" {
		t.Errorf("status field = %q, want ok", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestSelectRewardStats(t *testing.T) {
	h := New(newPolicy(t), quietLogger()).Handler()

	// UCB1 tries each arm once, in order, but it only learns from rewards, so
	// each selection is rewarded before the next one is requested.
	for want := range 3 {
		rec := do(h, "POST", "/select", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("select status = %d", rec.Code)
		}
		resp := decode[selectResponse](t, rec)
		if resp.Arm != want {
			t.Errorf("selection %d chose arm %d, want %d", want, resp.Arm, want)
		}
		if got := rec.Header().Get("X-Request-ID"); got != resp.RequestID || got == "" {
			t.Errorf("X-Request-ID %q does not match body id %q", got, resp.RequestID)
		}

		reward := "0"
		if want == 1 {
			reward = "1"
		}
		rec = do(h, "POST", "/reward", `{"request_id":"`+resp.RequestID+`","reward":`+reward+`}`)
		if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
			t.Fatalf("reward: status %d body %q, want 204 and no body", rec.Code, rec.Body.String())
		}
	}

	stats := decode[statsResponse](t, do(h, "GET", "/stats", ""))
	if stats.Policy != "ucb1" || stats.Selects != 3 || stats.Updates != 3 {
		t.Errorf("stats = %+v, want ucb1 with 3 selects and 3 updates", stats)
	}
	if got := stats.Arms[1]; got.Pulls != 1 || got.Mean != 1 {
		t.Errorf("arm 1 = %+v, want 1 pull with mean 1", got)
	}
}

// Feedback that has not arrived yet is invisible to the policy. Until the
// first reward comes back, a deterministic policy like UCB1 keeps choosing
// the same arm.
func TestSelectionsWithoutFeedbackRepeat(t *testing.T) {
	h := New(newPolicy(t), quietLogger()).Handler()
	for i := range 4 {
		if arm := decode[selectResponse](t, do(h, "POST", "/select", "")).Arm; arm != 0 {
			t.Fatalf("selection %d chose arm %d, want 0 while no reward has arrived", i, arm)
		}
	}
}

func TestRewardRejections(t *testing.T) {
	h := New(newPolicy(t), quietLogger()).Handler()
	id := decode[selectResponse](t, do(h, "POST", "/select", "")).RequestID
	if rec := do(h, "POST", "/reward", `{"request_id":"`+id+`","reward":0}`); rec.Code != http.StatusNoContent {
		t.Fatalf("setup: first reward status = %d", rec.Code)
	}

	tests := []struct {
		name string
		body string
		want int
	}{
		{"not json", `hello`, http.StatusBadRequest},
		{"empty body", ``, http.StatusBadRequest},
		{"unknown field", `{"request_id":"x","reward":1,"bonus":5}`, http.StatusBadRequest},
		{"two values", `{"request_id":"x","reward":1} {}`, http.StatusBadRequest},
		{"missing id", `{"reward":1}`, http.StatusBadRequest},
		{"missing reward", `{"request_id":"x"}`, http.StatusBadRequest},
		{"wrong type", `{"request_id":"x","reward":"high"}`, http.StatusBadRequest},
		{"reward too big", `{"request_id":"x","reward":1.5}`, http.StatusUnprocessableEntity},
		{"reward negative", `{"request_id":"x","reward":-0.1}`, http.StatusUnprocessableEntity},
		{"unknown id", `{"request_id":"nope","reward":1}`, http.StatusNotFound},
		{"already rewarded", `{"request_id":"` + id + `","reward":1}`, http.StatusConflict},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(h, "POST", "/reward", tc.body)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tc.want, rec.Body.String())
			}
			if msg := decode[errorBody](t, rec).Error; msg == "" {
				t.Error("error response has no message")
			}
		})
	}

	// None of the rejected requests may have reached the policy.
	stats := decode[statsResponse](t, do(h, "GET", "/stats", ""))
	if stats.Updates != 1 {
		t.Errorf("updates = %d, want only the one valid reward", stats.Updates)
	}
}

func TestBodyTooLarge(t *testing.T) {
	h := New(newPolicy(t), quietLogger()).Handler()
	big := `{"request_id":"` + strings.Repeat("a", maxBody) + `","reward":1}`
	if rec := do(h, "POST", "/reward", big); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", rec.Code)
	}
}

func TestRoutingErrors(t *testing.T) {
	h := New(newPolicy(t), quietLogger()).Handler()

	rec := do(h, "GET", "/select", "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /select status = %d, want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, "POST") {
		t.Errorf("Allow header = %q, want it to list POST", allow)
	}
	if rec := do(h, "GET", "/nowhere", ""); rec.Code != http.StatusNotFound {
		t.Errorf("GET /nowhere status = %d, want 404", rec.Code)
	}
}

func TestRequestIDsAreUnique(t *testing.T) {
	h := New(newPolicy(t), quietLogger()).Handler()
	seen := map[string]bool{}
	for range 200 {
		id := do(h, "GET", "/healthz", "").Header().Get("X-Request-ID")
		if id == "" || seen[id] {
			t.Fatalf("request id %q is empty or repeated", id)
		}
		seen[id] = true
	}
	// A client-supplied id is ignored.
	req := httptest.NewRequest("GET", "/healthz", nil)
	req.Header.Set("X-Request-ID", "chosen-by-client")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("X-Request-ID") == "chosen-by-client" {
		t.Error("server accepted a client-chosen request id")
	}
}

func TestLoggingRecordsTheRequest(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	h := New(newPolicy(t), log).Handler()

	rec := do(h, "POST", "/select", "")
	line := buf.String() // do() is synchronous, so the line is already written
	for _, want := range []string{"msg=request", "method=POST", "path=/select", "status=200", "id=" + rec.Header().Get("X-Request-ID")} {
		if !strings.Contains(line, want) {
			t.Errorf("log line %q is missing %q", line, want)
		}
	}
}

func TestRecoverPanic(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	boom := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("kaboom") })
	h := Chain(boom, requestID, logging(log), recoverPanic(log))

	rec := do(h, "GET", "/", "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if msg := decode[errorBody](t, rec).Error; strings.Contains(msg, "kaboom") {
		t.Errorf("error message %q leaks the panic value to the client", msg)
	}
	out := buf.String()
	if !strings.Contains(out, "panic in handler") || !strings.Contains(out, "kaboom") {
		t.Errorf("panic not logged: %q", out)
	}
	if !strings.Contains(out, "status=500") {
		t.Errorf("the request log should show status 500: %q", out)
	}
}

func TestChainOrder(t *testing.T) {
	var trace []string
	mw := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				trace = append(trace, name+" in")
				next.ServeHTTP(w, r)
				trace = append(trace, name+" out")
			})
		}
	}
	h := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { trace = append(trace, "handler") }),
		mw("a"), mw("b"))
	do(h, "GET", "/", "")
	want := []string{"a in", "b in", "handler", "b out", "a out"}
	if !slices.Equal(trace, want) {
		t.Errorf("trace = %v, want %v", trace, want)
	}
}

// End to end over real HTTP, with many clients at once. Run with -race.
func TestConcurrentClients(t *testing.T) {
	ts := httptest.NewServer(New(newPolicy(t), quietLogger()).Handler())
	defer ts.Close()

	const clients = 40
	var wg sync.WaitGroup
	errs := make(chan error, clients)
	for range clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Post(ts.URL+"/select", "application/json", nil)
			if err != nil {
				errs <- err
				return
			}
			var sel selectResponse
			err = json.NewDecoder(resp.Body).Decode(&sel)
			resp.Body.Close()
			if err != nil {
				errs <- err
				return
			}
			body := `{"request_id":"` + sel.RequestID + `","reward":1}`
			resp, err = http.Post(ts.URL+"/reward", "application/json", strings.NewReader(body))
			if err != nil {
				errs <- err
				return
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusNoContent {
				errs <- &statusError{resp.StatusCode}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	resp, err := http.Get(ts.URL + "/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var stats statsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}
	pulls := 0
	for _, a := range stats.Arms {
		pulls += a.Pulls
	}
	if stats.Selects != clients || stats.Updates != clients || pulls != clients {
		t.Errorf("stats = %+v (pulls %d), want %d of everything", stats, pulls, clients)
	}
}

type statusError struct{ code int }

func (e *statusError) Error() string { return "unexpected status " + http.StatusText(e.code) }

func TestSelectDelay(t *testing.T) {
	h := New(newPolicy(t), quietLogger(), WithSelectDelay(50*time.Millisecond)).Handler()
	start := time.Now()
	rec := do(h, "POST", "/select", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
		t.Errorf("select took %v, want at least the configured 50ms", elapsed)
	}
}

func TestSelectStopsWhenClientHangsUp(t *testing.T) {
	s := New(newPolicy(t), quietLogger(), WithSelectDelay(time.Hour))
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("POST", "/select", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		s.Handler().ServeHTTP(rec, req)
		close(done)
	}()
	cancel() // the client goes away
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler kept waiting after the request context was cancelled")
	}
	if rec.Code != 499 {
		t.Errorf("status = %d, want 499", rec.Code)
	}
	if selects, _ := s.policy.Calls(); selects != 0 {
		t.Errorf("policy was asked to select %d times for an abandoned request", selects)
	}
}

func TestDrainingFailsHealthChecks(t *testing.T) {
	s := New(newPolicy(t), quietLogger())
	h := s.Handler()
	if rec := do(h, "GET", "/healthz", ""); rec.Code != http.StatusOK {
		t.Fatalf("before draining: status = %d", rec.Code)
	}
	s.SetDraining()
	rec := do(h, "GET", "/healthz", "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("while draining: status = %d, want 503", rec.Code)
	}
	if got := decode[map[string]string](t, rec)["status"]; got != "draining" {
		t.Errorf("status field = %q, want draining", got)
	}
}
