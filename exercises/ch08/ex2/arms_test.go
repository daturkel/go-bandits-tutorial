package ex2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testStats() []ArmStat {
	return []ArmStat{{0, 10, 0.2}, {1, 30, 0.6}, {2, 5, 0.4}}
}

func get(h http.Handler, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestList(t *testing.T) {
	rec := get(Routes(testStats), "GET", "/arms")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got []ArmStat
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || len(got) != 3 || got[1].Pulls != 30 {
		t.Errorf("body %q decoded to %v (err %v)", rec.Body.String(), got, err)
	}
}

func TestOneArm(t *testing.T) {
	rec := get(Routes(testStats), "GET", "/arms/1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got ArmStat
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got != (ArmStat{1, 30, 0.6}) {
		t.Errorf("body %q decoded to %+v (err %v)", rec.Body.String(), got, err)
	}
}

func TestBadArm(t *testing.T) {
	tests := []struct {
		path string
		want int
	}{
		{"/arms/abc", http.StatusBadRequest},
		{"/arms/1.5", http.StatusBadRequest},
		{"/arms/3", http.StatusNotFound},
		{"/arms/-1", http.StatusNotFound},
		{"/arms/99", http.StatusNotFound},
	}
	for _, tc := range tests {
		if rec := get(Routes(testStats), "GET", tc.path); rec.Code != tc.want {
			t.Errorf("GET %s: status = %d, want %d", tc.path, rec.Code, tc.want)
		}
	}
}

func TestOtherMethods(t *testing.T) {
	for _, path := range []string{"/arms", "/arms/1"} {
		rec := get(Routes(testStats), "POST", path)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s: status = %d, want 405", path, rec.Code)
		}
	}
}

func TestStatsCalledPerRequest(t *testing.T) {
	calls := 0
	h := Routes(func() []ArmStat {
		calls++
		return testStats()
	})
	get(h, "GET", "/arms")
	get(h, "GET", "/arms/0")
	if calls != 2 {
		t.Errorf("stats() called %d times for 2 requests, want 2 (fresh data every time)", calls)
	}
}
