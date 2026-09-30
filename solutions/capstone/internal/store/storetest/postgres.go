package storetest

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
)

// DatabaseURL returns the PostgreSQL URL for integration tests, taken from
// BANDIT_TEST_DATABASE_URL (tools/pg.sh start prints a suitable one). If it is
// not set the test is skipped, so `go test ./...` works on any machine.
func DatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("BANDIT_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("BANDIT_TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}
	return url
}

var schemaSeq atomic.Int64

// NewSchema creates an empty schema (a namespace inside the database) for one
// test and drops it when the test ends, so tests never see each other's
// tables. Pass the returned name to store.WithSchema.
func NewSchema(t *testing.T, url string) string {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	name := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '_'
	}, strings.ToLower(t.Name()))
	if len(name) > 40 {
		name = name[:40]
	}
	name = fmt.Sprintf("t%d_%d_%s", os.Getpid(), schemaSeq.Add(1), name)
	if _, err := conn.Exec(ctx, "CREATE SCHEMA "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Exec(context.Background(), "DROP SCHEMA "+name+" CASCADE")
		conn.Close(context.Background())
	})
	return name
}
