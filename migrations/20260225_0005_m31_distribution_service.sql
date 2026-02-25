-- M31-Distribution-Service initial schema.
-- Ownership: distribution-service (single-writer logical ownership).

CREATE TABLE IF NOT EXISTS distribution_items (
    id UUID PRIMARY KEY,
    influencer_id UUID NOT NULL,
    clip_id UUID NOT NULL,
    campaign_id UUID NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'claimed',
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    claim_expires_at TIMESTAMPTZ NOT NULL,
    scheduled_for_utc TIMESTAMPTZ NULL,
    timezone VARCHAR(64) NULL,
    platforms TEXT[] NOT NULL DEFAULT '{}',
    caption_text TEXT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NULL,
    published_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT distribution_items_status_check
        CHECK (status IN ('claimed', 'scheduled', 'publishing', 'published', 'failed', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_distribution_items_influencer_id ON distribution_items (influencer_id, claimed_at DESC);
CREATE INDEX IF NOT EXISTS idx_distribution_items_campaign_id ON distribution_items (campaign_id);
CREATE INDEX IF NOT EXISTS idx_distribution_items_status ON distribution_items (status);

CREATE TABLE IF NOT EXISTS distribution_captions (
    id UUID PRIMARY KEY,
    distribution_item_id UUID NOT NULL REFERENCES distribution_items (id) ON DELETE CASCADE,
    platform VARCHAR(50) NULL,
    caption_text TEXT NOT NULL,
    hashtags TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_distribution_captions_item_id
    ON distribution_captions (distribution_item_id);

CREATE TABLE IF NOT EXISTS distribution_overlays (
    id UUID PRIMARY KEY,
    distribution_item_id UUID NOT NULL REFERENCES distribution_items (id) ON DELETE CASCADE,
    overlay_type VARCHAR(20) NOT NULL,
    asset_path TEXT NOT NULL,
    duration_seconds DECIMAL(5, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT distribution_overlays_type_check CHECK (overlay_type IN ('intro', 'outro'))
);

CREATE INDEX IF NOT EXISTS idx_distribution_overlays_item_id
    ON distribution_overlays (distribution_item_id);

CREATE TABLE IF NOT EXISTS distribution_platform_status (
    id UUID PRIMARY KEY,
    distribution_item_id UUID NOT NULL REFERENCES distribution_items (id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    status VARCHAR(24) NOT NULL,
    platform_post_url TEXT NULL,
    error_message TEXT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT distribution_platform_status_state_check
        CHECK (status IN ('pending', 'publishing', 'published', 'failed'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_distribution_platform_status_unique
    ON distribution_platform_status (distribution_item_id, platform);

CREATE TABLE IF NOT EXISTS publishing_analytics (
    id UUID PRIMARY KEY,
    distribution_item_id UUID NOT NULL REFERENCES distribution_items (id) ON DELETE CASCADE,
    influencer_id UUID NOT NULL,
    campaign_id UUID NOT NULL,
    platform VARCHAR(50) NOT NULL,
    success BOOLEAN NOT NULL,
    error_code VARCHAR(50) NULL,
    error_message TEXT NULL,
    time_to_publish_seconds INTEGER NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_publishing_analytics_distribution_item_id
    ON publishing_analytics (distribution_item_id, created_at DESC);

