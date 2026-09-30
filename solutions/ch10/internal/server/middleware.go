package server

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// Middleware wraps a handler with behaviour that runs before and after it.
type Middleware func(http.Handler) http.Handler

// Chain wraps h in mws. The first middleware listed is the outermost: it sees
// the request first and the response last.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// ctxKey is unexported, so no other package can collide with our context keys.
type ctxKey struct{}

// RequestID returns the id the requestID middleware attached to ctx, or "".
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// requestID gives every request a fresh random id, exposes it in the
// X-Request-ID response header, and stores it in the request context. Ids are
// always generated here; a client-supplied one is ignored, so callers cannot
// pick ids that collide.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := rand.Text()
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), ctxKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// statusRecorder remembers what the handler wrote. It embeds the real
// ResponseWriter, so everything it does not override passes straight through.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK // Write without WriteHeader means 200
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// logging writes one structured line per request, after it completes.
func logging(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			if rec.status == 0 {
				rec.status = http.StatusOK // handler wrote nothing at all
			}
			log.Info("request",
				"id", RequestID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"bytes", rec.bytes,
				"duration", time.Since(start),
			)
		})
	}
}

// recoverPanic turns a panic in a handler into a 500 response, so one bad
// request cannot take the process down. net/http already recovers panics per
// connection, but it just closes the connection with no response and logs to
// a different place.
func recoverPanic(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				if v == http.ErrAbortHandler {
					panic(v) // net/http's deliberate "abort quietly" signal: pass it on
				}
				log.Error("panic in handler",
					"id", RequestID(r.Context()),
					"panic", v,
					"stack", string(debug.Stack()),
				)
				writeError(w, http.StatusInternalServerError, "internal error")
			}()
			next.ServeHTTP(w, r)
		})
	}
}
