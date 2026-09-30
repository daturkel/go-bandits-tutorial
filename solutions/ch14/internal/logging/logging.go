// Package logging builds the process's structured logger.
package logging

import (
	"fmt"
	"io"
	"log/slog"
)

// New returns a logger writing to w in the given format ("text" or "json"),
// dropping records below level.
func New(w io.Writer, format string, level slog.Level) (*slog.Logger, error) {
	opts := &slog.HandlerOptions{Level: level}
	switch format {
	case "text":
		return slog.New(slog.NewTextHandler(w, opts)), nil
	case "json":
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	default:
		return nil, fmt.Errorf("unknown log format %q (want text or json)", format)
	}
}
