-- M09-Content-Library-Marketplace initial schema.
-- Ownership: content-marketplace-service (single-writer logical ownership).

CREATE TABLE IF NOT EXISTS clips (
    clip_id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL,
    submission_id UUID NOT NULL,
    creator_id UUID NOT NULL,
    title VARCHAR(120),
    description TEXT,
    niche VARCHAR(40) NOT NULL,
    duration_seconds INTEGER NOT NULL,
    preview_url TEXT NOT NULL,
    download_asset_id UUID NOT NULL,
    exclusivity VARCHAR(20) NOT NULL DEFAULT 'non_exclusive',
    claim_limit INTEGER NOT NULL DEFAULT 50,
    views_7d INTEGER NOT NULL DEFAULT 0,
    votes_7d INTEGER NOT NULL DEFAULT 0,
    engagement_rate DECIMAL(6,3) NOT NULL DEFAULT 0.000,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT clips_exclusivity_check CHECK (exclusivity IN ('exclusive', 'non_exclusive')),
    CONSTRAINT clips_status_check CHECK (status IN ('active', 'paused', 'archived'))
);

CREATE INDEX IF NOT EXISTS idx_clips_campaign_id ON clips (campaign_id);
CREATE INDEX IF NOT EXISTS idx_clips_niche ON clips (niche);
CREATE INDEX IF NOT EXISTS idx_clips_status ON clips (status);

CREATE TABLE IF NOT EXISTS clip_claims (
    claim_id UUID PRIMARY KEY,
    clip_id UUID NOT NULL REFERENCES clips (clip_id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    claim_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    request_id UUID NOT NULL,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT clip_claims_claim_type_check CHECK (claim_type IN ('exclusive', 'non_exclusive')),
    CONSTRAINT clip_claims_status_check CHECK (status IN ('active', 'published', 'paid', 'expired', 'cancelled', 'failed')),
    CONSTRAINT clip_claims_unique_request UNIQUE (request_id)
);

CREATE INDEX IF NOT EXISTS idx_clip_claims_clip_id ON clip_claims (clip_id);
CREATE INDEX IF NOT EXISTS idx_clip_claims_user_id ON clip_claims (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_clip_claims_clip_user_unique ON clip_claims (clip_id, user_id);

CREATE TABLE IF NOT EXISTS clip_downloads (
    download_id UUID PRIMARY KEY,
    clip_id UUID NOT NULL REFERENCES clips (clip_id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    ip_address INET,
    user_agent TEXT,
    downloaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_clip_download_user ON clip_downloads (user_id);
CREATE INDEX IF NOT EXISTS idx_clip_download_clip ON clip_downloads (clip_id);

CREATE TABLE IF NOT EXISTS content_marketplace_idempotency (
    key TEXT PRIMARY KEY,
    request_hash TEXT NOT NULL,
    claim_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS content_marketplace_outbox (
    outbox_id UUID PRIMARY KEY,
    event_type TEXT NOT NULL,
    partition_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_content_marketplace_outbox_status_created_at
    ON content_marketplace_outbox (status, created_at);
