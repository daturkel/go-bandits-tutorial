package ex1

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// BearerAuth returns middleware that lets a request through only if it carries
//
//	Authorization: Bearer <token>
//
// with exactly the given token. Otherwise it must not call next, and instead
// responds 401 with the header `WWW-Authenticate: Bearer`, a
// `Content-Type: application/json` header, and the body {"error":"unauthorized"}.
//
// Compare the token with crypto/subtle.ConstantTimeCompare, so response time
// does not reveal how much of a guess was right.
func BearerAuth(token string) func(http.Handler) http.Handler {
	want := []byte(token)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || subtle.ConstantTimeCompare([]byte(got), want) != 1 {
				w.Header().Set("WWW-Authenticate", "Bearer")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}` + "\n"))
				return // never reaches next
			}
			next.ServeHTTP(w, r)
		})
	}
}
