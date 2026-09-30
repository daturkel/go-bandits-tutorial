package store

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"banditlab/internal/bandit"
)

//go:embed schema.sql
var schema string // the file's contents, compiled into the binary

// schemaLockID is an arbitrary number that identifies "setting up banditlab's
// tables" to pg_advisory_xact_lock.
const schemaLockID = 0x62616e64

// Postgres is a Store backed by PostgreSQL. Every timestamp comparison uses
// the database's clock, so replicas with drifting clocks agree on expiry.
type Postgres struct {
	pool *pgxpool.Pool
}

// PostgresOption adjusts OpenPostgres.
type PostgresOption func(*pgxpool.Config)

// WithSchema makes the connection use the named schema (namespace) instead of
// the default. The schema must already exist. Integration tests use one
// schema per test so they cannot see each other's rows.
func WithSchema(name string) PostgresOption {
	return func(cfg *pgxpool.Config) {
		cfg.ConnConfig.RuntimeParams["search_path"] = name
	}
}

// OpenPostgres connects to the database at url, creates the tables if they
// are missing, and makes sure there is a totals row for each of arms arms.
// It fails if the database already holds more arms than that, since silently
// dropping learned data is worse than refusing to start.
func OpenPostgres(ctx context.Context, url string, arms int, opts ...PostgresOption) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("database url: %w", err)
	}
	for _, opt := range opts {
		opt(cfg)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	p := &Postgres{pool: pool}
	if err := p.init(ctx, arms); err != nil {
		pool.Close()
		return nil, err
	}
	return p, nil
}

func (p *Postgres) init(ctx context.Context, arms int) error {
	if err := p.pool.Ping(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	// Replicas start at the same time against the same database, and
	// CREATE TABLE IF NOT EXISTS is not safe to run concurrently: two sessions
	// can both find the table missing and one fails on the catalog's unique
	// index. An advisory lock makes them take turns; it is released when the
	// transaction ends.
	err := pgx.BeginFunc(ctx, p.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, schemaLockID); err != nil {
			return fmt.Errorf("lock: %w", err)
		}
		// Several statements in one Exec: pgx sends them as a simple query.
		if _, err := tx.Exec(ctx, schema); err != nil {
			return fmt.Errorf("create tables: %w", err)
		}
		// Add rows for new arms; leave existing rows (and their data) alone.
		if _, err := tx.Exec(ctx,
			`INSERT INTO arm_totals (arm) SELECT g FROM generate_series(0, $1::int - 1) AS g ON CONFLICT DO NOTHING`, arms); err != nil {
			return fmt.Errorf("create arm rows: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	var have int
	if err := p.pool.QueryRow(ctx, `SELECT count(*) FROM arm_totals`).Scan(&have); err != nil {
		return err
	}
	if have != arms {
		return fmt.Errorf("database holds %d arms but %d are configured", have, arms)
	}
	return nil
}

func (p *Postgres) AddPending(ctx context.Context, id string, arm int, ttl time.Duration) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO pending (request_id, arm, expires_at) VALUES ($1, $2, now() + make_interval(secs => $3))`,
		id, arm, ttl.Seconds())
	return err
}

// claimAndCount is one statement, so it is atomic without an explicit
// transaction: the pending row is marked rewarded and the arm's totals are
// incremented together or not at all. The WHERE clause is the whole
// correctness argument. A row can only be claimed while it is unrewarded and
// unexpired, and PostgreSQL makes concurrent UPDATEs of one row take turns.
const claimAndCount = `
WITH claimed AS (
    UPDATE pending SET rewarded = true
    WHERE request_id = $1 AND NOT rewarded AND expires_at > now()
    RETURNING arm
)
UPDATE arm_totals t
SET pulls = t.pulls + 1, reward_sum = t.reward_sum + $2
FROM claimed
WHERE t.arm = claimed.arm
RETURNING t.arm`

func (p *Postgres) Reward(ctx context.Context, id string, reward float64) (int, error) {
	var arm int
	err := p.pool.QueryRow(ctx, claimAndCount, id, reward).Scan(&arm)
	if err == nil {
		return arm, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	// Nothing was claimed. Find out why, so the caller gets the right error.
	var rewarded bool
	err = p.pool.QueryRow(ctx,
		`SELECT rewarded FROM pending WHERE request_id = $1 AND expires_at > now()`, id).Scan(&rewarded)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return 0, ErrUnknownID
	case err != nil:
		return 0, err
	case rewarded:
		return 0, ErrAlreadyRewarded
	default:
		// Unrewarded and unexpired, yet the claim matched nothing. The foreign
		// key on pending.arm makes this impossible, so report it loudly.
		return 0, fmt.Errorf("pending row %q could not be claimed: inconsistent state", id)
	}
}

// Expire deletes expired rows and, optionally, counts the unrewarded ones.
// It takes several steps that must succeed together, so it runs in an
// explicit transaction: pgx.BeginFunc commits if the function returns nil and
// rolls back if it returns an error or panics.
func (p *Postgres) Expire(ctx context.Context, implicitReward *float64) (map[int]int, error) {
	expired := map[int]int{}
	err := pgx.BeginFunc(ctx, p.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `DELETE FROM pending WHERE expires_at <= now() RETURNING arm, rewarded`)
		if err != nil {
			return err
		}
		type gone struct {
			Arm      int
			Rewarded bool
		}
		removed, err := pgx.CollectRows(rows, pgx.RowToStructByPos[gone])
		if err != nil {
			return err
		}
		for _, g := range removed {
			if !g.Rewarded {
				expired[g.Arm]++
			}
		}
		if implicitReward == nil {
			return nil
		}
		for arm, n := range expired {
			if _, err := tx.Exec(ctx,
				`UPDATE arm_totals SET pulls = pulls + $2, reward_sum = reward_sum + $2 * $3 WHERE arm = $1`,
				arm, n, *implicitReward); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return expired, nil
}

func (p *Postgres) Totals(ctx context.Context) ([]bandit.ArmTotals, error) {
	rows, err := p.pool.Query(ctx, `SELECT arm, pulls, reward_sum FROM arm_totals ORDER BY arm`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[bandit.ArmTotals])
}

func (p *Postgres) PendingCounts(ctx context.Context) ([]int, error) {
	// LEFT JOIN so that arms with nothing pending still get a row (with 0).
	rows, err := p.pool.Query(ctx, `
		SELECT count(p.request_id)
		FROM arm_totals a
		LEFT JOIN pending p ON p.arm = a.arm AND NOT p.rewarded AND p.expires_at > now()
		GROUP BY a.arm
		ORDER BY a.arm`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int])
}

func (p *Postgres) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }

func (p *Postgres) Close() error {
	p.pool.Close()
	return nil
}

var _ Store = (*Postgres)(nil)
