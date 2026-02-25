-- M08-Voting-Engine reliability and schema parity additions.
-- Safe additive migration for legacy M27 tables used by M08 successor runtime.

ALTER TABLE votes
    ADD COLUMN IF NOT EXISTS round_id UUID NULL REFERENCES voting_rounds (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS reputation_score_snapshot DECIMAL(5, 2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS ip_address TEXT NULL,
    ADD COLUMN IF NOT EXISTS user_agent TEXT NULL;

DROP INDEX IF EXISTS idx_votes_unique_user_submission;
CREATE UNIQUE INDEX IF NOT EXISTS idx_votes_unique_user_submission_no_round
    ON votes (user_id, submission_id)
    WHERE round_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_votes_unique_user_submission_round
    ON votes (user_id, submission_id, round_id)
    WHERE round_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_votes_round_id ON votes (round_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_votes_submission_active ON votes (submission_id, retracted, created_at DESC);

CREATE TABLE IF NOT EXISTS voting_outbox (
    outbox_id UUID PRIMARY KEY,
    event_type TEXT NOT NULL,
    partition_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_voting_outbox_status_created
    ON voting_outbox (status, created_at ASC);

CREATE TABLE IF NOT EXISTS voting_event_dedup (
    event_id UUID PRIMARY KEY,
    payload_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_voting_event_dedup_expires_at
    ON voting_event_dedup (expires_at ASC);
