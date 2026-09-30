// Package store keeps what the service must not lose: per-arm totals and the
// selections still waiting for a reward.
package store

import (
	"context"
	"errors"
	"time"

	"banditlab/internal/bandit"
)

var (
	// ErrUnknownID means no selection with this id is waiting for a reward:
	// it never existed, or it expired and was removed.
	ErrUnknownID = errors.New("unknown or expired request id")

	// ErrAlreadyRewarded means the id's reward was already recorded.
	ErrAlreadyRewarded = errors.New("reward already recorded for this request id")
)

// Store is the service's durable state. Implementations must be safe for
// concurrent use, and each method must be atomic: after a crash or a
// concurrent call, no method's effect is ever half applied.
type Store interface {
	// AddPending records that the selection id returned arm, valid for ttl.
	// Adding an id that already exists is an error.
	AddPending(ctx context.Context, id string, arm int, ttl time.Duration) error

	// Reward claims the pending selection id and adds reward to that arm's
	// totals, as one atomic step. It returns the arm. Of any number of
	// concurrent calls for the same id, exactly one succeeds; the others get
	// ErrAlreadyRewarded. An id that is unknown or past its ttl gets
	// ErrUnknownID, and nothing changes.
	Reward(ctx context.Context, id string, reward float64) (arm int, err error)

	// Expire removes every pending selection whose ttl has passed. If
	// implicitReward is non-nil, each removed selection that never received
	// a reward is also counted, as if it had been rewarded with that value
	// (for example 0: no conversion within the window). It returns, per arm,
	// how many such unrewarded selections expired.
	Expire(ctx context.Context, implicitReward *float64) (expired map[int]int, err error)

	// Totals returns one entry per arm, in arm order.
	Totals(ctx context.Context) ([]bandit.ArmTotals, error)

	// Ping reports whether the store is reachable.
	Ping(ctx context.Context) error

	// Close releases the store's resources.
	Close() error
}
