package ex2

import (
	"io"
	"log/slog"
)

// NewRedactingLogger returns a logger that writes JSON records to w, with the
// value of every attribute whose key is in keys replaced by the string
// "[REDACTED]". It must apply to attributes at any depth, including those
// inside slog.Group, and to attributes added with logger.With.
//
// Do not write a slog.Handler from scratch: slog.HandlerOptions has a
// ReplaceAttr hook that sees every attribute just before it is written.
func NewRedactingLogger(w io.Writer, keys ...string) *slog.Logger {
	// TODO
	return slog.New(slog.NewJSONHandler(w, nil))
}
