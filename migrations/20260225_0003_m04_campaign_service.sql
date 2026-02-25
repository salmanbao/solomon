-- M04-Campaign-Service initial schema.
-- Ownership: campaign-service (single-writer logical ownership).

CREATE TABLE IF NOT EXISTS campaigns (
    campaign_id UUID PRIMARY KEY,
    brand_id UUID NOT NULL,
    title VARCHAR(120) NOT NULL,
    description TEXT NOT NULL,
    instructions TEXT NOT NULL,
    niche VARCHAR(64) NOT NULL,
    allowed_platforms TEXT[] NOT NULL DEFAULT '{}',
    required_hashtags TEXT[] NOT NULL DEFAULT '{}',
    budget_total DECIMAL(12, 2) NOT NULL,
    budget_spent DECIMAL(12, 2) NOT NULL DEFAULT 0,
    budget_remaining DECIMAL(12, 2) NOT NULL DEFAULT 0,
    rate_per_1k_views DECIMAL(8, 4) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    launched_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL,
    CONSTRAINT campaigns_status_check CHECK (status IN ('draft', 'active', 'paused', 'completed'))
);

CREATE INDEX IF NOT EXISTS idx_campaigns_brand_id ON campaigns (brand_id);
CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns (status);
CREATE INDEX IF NOT EXISTS idx_campaigns_created_at ON campaigns (created_at DESC);

CREATE TABLE IF NOT EXISTS campaign_media (
    media_id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES campaigns (campaign_id) ON DELETE CASCADE,
    asset_path TEXT NOT NULL,
    content_type VARCHAR(120) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ready',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT campaign_media_status_check CHECK (status IN ('uploading', 'processing', 'ready', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_campaign_media_campaign_id ON campaign_media (campaign_id);

CREATE TABLE IF NOT EXISTS campaign_budget_log (
    log_id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES campaigns (campaign_id) ON DELETE CASCADE,
    amount_delta DECIMAL(12, 2) NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_budget_log_campaign_id ON campaign_budget_log (campaign_id, created_at DESC);

CREATE TABLE IF NOT EXISTS campaign_state_history (
    history_id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES campaigns (campaign_id) ON DELETE CASCADE,
    from_state VARCHAR(20) NOT NULL,
    to_state VARCHAR(20) NOT NULL,
    changed_by UUID NOT NULL,
    change_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_state_history_campaign_id ON campaign_state_history (campaign_id, created_at DESC);

CREATE TABLE IF NOT EXISTS campaign_idempotency (
    key TEXT PRIMARY KEY,
    request_hash TEXT NOT NULL,
    response_payload JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

