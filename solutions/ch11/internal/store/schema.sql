-- Per-arm totals: enough to rebuild any policy, and safe to add across replicas.
CREATE TABLE IF NOT EXISTS arm_totals (
    arm        integer          PRIMARY KEY,
    pulls      bigint           NOT NULL DEFAULT 0,
    reward_sum double precision NOT NULL DEFAULT 0
);

-- Selections waiting for a reward. rewarded stays true until the row expires, so
-- a repeated reward can be told apart from an unknown id.
CREATE TABLE IF NOT EXISTS pending (
    request_id text        PRIMARY KEY,
    arm        integer     NOT NULL REFERENCES arm_totals (arm),
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    rewarded   boolean     NOT NULL DEFAULT false
);

-- Lets the expiry sweep find old rows without scanning the table.
CREATE INDEX IF NOT EXISTS pending_expires_at ON pending (expires_at);
