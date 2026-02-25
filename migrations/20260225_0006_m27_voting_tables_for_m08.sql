-- M27-Voting-Engine legacy schema used by M08 successor in MVP.
-- Ownership remains legacy table ids (schema-only dependency style).

CREATE TABLE IF NOT EXISTS voting_rounds (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT voting_rounds_status_check CHECK (status IN ('scheduled', 'active', 'closing_soon', 'closed', 'archived'))
);

CREATE INDEX IF NOT EXISTS idx_voting_rounds_campaign_id ON voting_rounds (campaign_id);
CREATE INDEX IF NOT EXISTS idx_voting_rounds_status ON voting_rounds (status);

CREATE TABLE IF NOT EXISTS votes (
    id UUID PRIMARY KEY,
    submission_id UUID NOT NULL,
    campaign_id UUID NOT NULL,
    user_id UUID NOT NULL,
    vote_type VARCHAR(20) NOT NULL,
    weight DECIMAL(8, 3) NOT NULL DEFAULT 1.0,
    retracted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT votes_vote_type_check CHECK (vote_type IN ('upvote', 'downvote'))
);

CREATE INDEX IF NOT EXISTS idx_votes_submission_id ON votes (submission_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_votes_campaign_id ON votes (campaign_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_votes_unique_user_submission
    ON votes (user_id, submission_id);

CREATE TABLE IF NOT EXISTS vote_quarantine (
    id UUID PRIMARY KEY,
    vote_id UUID NOT NULL REFERENCES votes (id) ON DELETE CASCADE,
    risk_score DECIMAL(4, 3) NOT NULL,
    reason TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending_review',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT vote_quarantine_status_check CHECK (status IN ('pending_review', 'approved', 'rejected'))
);

CREATE INDEX IF NOT EXISTS idx_vote_quarantine_vote_id ON vote_quarantine (vote_id);
CREATE INDEX IF NOT EXISTS idx_vote_quarantine_status ON vote_quarantine (status);

CREATE TABLE IF NOT EXISTS voting_engine_idempotency (
    key TEXT PRIMARY KEY,
    request_hash TEXT NOT NULL,
    vote_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

