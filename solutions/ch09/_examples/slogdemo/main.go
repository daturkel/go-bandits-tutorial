package main

import (
	"context"
	"log/slog"
	"os"
)

// noTime drops the timestamp so the output below is the same on every run.
func noTime(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		return slog.Attr{}
	}
	return a
}

func demo(log *slog.Logger) {
	log.Debug("cache warmed") // below the default level: not printed
	log.Info("request", "method", "POST", "path", "/select", "status", 200)
	log.Warn("slow request", "duration_ms", 1520.5)

	// With returns a logger that adds attributes to every record.
	reqLog := log.With("request_id", "abc123")
	reqLog.Info("selected", "arm", 2)

	// Group nests related attributes.
	log.Info("config", slog.Group("server", "addr", ":8080", "arms", 3))

	// Errors are ordinary values.
	log.Error("reward failed", "err", os.ErrNotExist)

	// The Context variants pass a context to the handler, which a custom
	// handler can use to add values it carries.
	log.InfoContext(context.Background(), "done")
}

func main() {
	opts := &slog.HandlerOptions{ReplaceAttr: noTime}

	println("--- text handler ---")
	demo(slog.New(slog.NewTextHandler(os.Stdout, opts)))

	println("--- json handler ---")
	demo(slog.New(slog.NewJSONHandler(os.Stdout, opts)))
}
