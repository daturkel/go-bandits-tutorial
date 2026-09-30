package ex1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBearerAuth(t *testing.T) {
	var called int
	protected := BearerAuth("s3cret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusTeapot) // an unmistakable "the inner handler ran"
	}))

	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"correct token", "Bearer s3cret", http.StatusTeapot},
		{"no header", "", http.StatusUnauthorized},
		{"wrong token", "Bearer nope", http.StatusUnauthorized},
		{"token is only a prefix", "Bearer s3cre", http.StatusUnauthorized},
		{"token with extra", "Bearer s3cret!", http.StatusUnauthorized},
		{"wrong scheme", "Basic s3cret", http.StatusUnauthorized},
		{"no scheme", "s3cret", http.StatusUnauthorized},
		{"lowercase scheme", "bearer s3cret", http.StatusUnauthorized},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before := called
			req := httptest.NewRequest("GET", "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			protected.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
			if tc.want == http.StatusUnauthorized {
				if called != before {
					t.Error("the inner handler ran for an unauthorised request")
				}
				if got := rec.Header().Get("WWW-Authenticate"); got != "Bearer" {
					t.Errorf("WWW-Authenticate = %q, want Bearer", got)
				}
				if got := rec.Header().Get("Content-Type"); got != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", got)
				}
				if got := strings.TrimSpace(rec.Body.String()); got != `{"error":"unauthorized"}` {
					t.Errorf("body = %q", got)
				}
			}
		})
	}
}
