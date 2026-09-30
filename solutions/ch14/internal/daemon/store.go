package daemon

import (
	"context"
	"fmt"
	"log/slog"

	"banditlab/internal/store"
)

// OpenStore connects to PostgreSQL if url is set and otherwise keeps state in
// memory. An in-memory store is private to its process, so separate daemons
// cannot share one: it is for trying a single daemon out.
func OpenStore(ctx context.Context, url string, arms int, log *slog.Logger) (store.Store, error) {
	if url == "" {
		log.Warn("no database configured: state is kept in memory, private to this process, and lost on exit")
		return store.NewMemory(arms), nil
	}
	st, err := store.OpenPostgres(ctx, url, arms)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	log.Info("database connected")
	return st, nil
}
