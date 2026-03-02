-- M23-Campaign-Discovery-Service canonical ownership tables.

CREATE TABLE IF NOT EXISTS campaign_eligibility_cache (
    eligibility_id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL,
    user_id UUID NOT NULL,
    is_eligible BOOLEAN NOT NULL,
    reason TEXT NULL,
    cached_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_campaign_eligibility_cache_campaign_user
    ON campaign_eligibility_cache (campaign_id, user_id);
CREATE INDEX IF NOT EXISTS idx_campaign_eligibility_cache_expires_at
    ON campaign_eligibility_cache (expires_at);

CREATE TABLE IF NOT EXISTS campaign_ranking_scores (
    ranking_id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL UNIQUE,
    popularity_score NUMERIC(5, 2) NOT NULL DEFAULT 0,
    budget_score NUMERIC(5, 2) NOT NULL DEFAULT 0,
    freshness_score NUMERIC(5, 2) NOT NULL DEFAULT 0,
    trending_score NUMERIC(5, 2) NOT NULL DEFAULT 0,
    combined_score NUMERIC(6, 2) NOT NULL DEFAULT 0,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_campaign_ranking_scores_combined
    ON campaign_ranking_scores (combined_score DESC);

CREATE TABLE IF NOT EXISTS discover_audit_log (
    audit_id UUID PRIMARY KEY,
    user_id UUID NULL,
    action TEXT NOT NULL,
    campaign_id UUID NULL,
    filter_params JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_count INTEGER NULL,
    execution_time_ms INTEGER NULL,
    ip_address TEXT NULL,
    user_agent TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_discover_audit_log_user_created
    ON discover_audit_log (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_discover_audit_log_campaign_created
    ON discover_audit_log (campaign_id, created_at DESC);

CREATE TABLE IF NOT EXISTS featured_placements (
    placement_id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL,
    placement_start TIMESTAMPTZ NOT NULL,
    placement_end TIMESTAMPTZ NOT NULL,
    boost_score INTEGER NOT NULL DEFAULT 100,
    paid_by_user_id UUID NOT NULL,
    feature_type VARCHAR(32) NOT NULL,
    placement_cost NUMERIC(12, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT featured_placements_dates_check CHECK (placement_start < placement_end),
    CONSTRAINT featured_placements_type_check CHECK (feature_type IN ('sponsored', 'paid'))
);
CREATE INDEX IF NOT EXISTS idx_featured_placements_campaign_active
    ON featured_placements (campaign_id, placement_end DESC);

CREATE TABLE IF NOT EXISTS user_bookmarks (
    bookmark_id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    campaign_id UUID NOT NULL,
    tag TEXT NULL,
    note TEXT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_bookmarks_status_check CHECK (status IN ('active', 'hidden'))
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_user_bookmarks_user_campaign
    ON user_bookmarks (user_id, campaign_id);
CREATE INDEX IF NOT EXISTS idx_user_bookmarks_user_created
    ON user_bookmarks (user_id, created_at DESC);

