package ex1

import "net/http"

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
	return func(next http.Handler) http.Handler {
		// TODO
		return next
	}
}
