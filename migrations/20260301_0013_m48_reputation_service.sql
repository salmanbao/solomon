-- M48-Reputation-Service canonical ownership tables and reliability controls.

CREATE TABLE IF NOT EXISTS badges (
    badge_id VARCHAR(64) PRIMARY KEY,
    badge_name TEXT NOT NULL,
    category TEXT NOT NULL,
    rarity TEXT NOT NULL,
    criteria_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    permanent_flag BOOLEAN NOT NULL DEFAULT false,
    icon_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_badges (
    user_badge_id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    badge_id VARCHAR(64) NOT NULL,
    earned_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NULL,
    public BOOLEAN NOT NULL DEFAULT true
);
CREATE INDEX IF NOT EXISTS idx_user_badges_user_id ON user_badges (user_id);
CREATE INDEX IF NOT EXISTS idx_user_badges_badge_id ON user_badges (badge_id);

CREATE TABLE IF NOT EXISTS user_reputation_scores (
    user_id VARCHAR(64) PRIMARY KEY,
    overall_score DECIMAL(5, 2) NOT NULL DEFAULT 0,
    current_score INTEGER NOT NULL DEFAULT 0,
    previous_score INTEGER NOT NULL DEFAULT 0,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    next_recalculation_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_date DATE NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_user_reputation_scores_overall_score
    ON user_reputation_scores (overall_score DESC);

CREATE TABLE IF NOT EXISTS user_reputation_tiers (
    user_id VARCHAR(64) PRIMARY KEY,
    current_tier VARCHAR(16) NOT NULL,
    tier_since TIMESTAMPTZ NOT NULL,
    promoted_at TIMESTAMPTZ NULL,
    demoted_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS user_reputation_signals (
    signal_id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    signal_type TEXT NOT NULL,
    signal_value NUMERIC(12, 4) NOT NULL,
    source_event_id VARCHAR(128) NOT NULL,
    signal_timestamp TIMESTAMPTZ NOT NULL,
    data_source TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_user_reputation_signals_idempotent
    ON user_reputation_signals (user_id, signal_type, source_event_id);

CREATE TABLE IF NOT EXISTS user_signal_snapshot (
    snapshot_id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    snapshot_date DATE NOT NULL,
    all_signals JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_user_signal_snapshot_user_date
    ON user_signal_snapshot (user_id, snapshot_date);

CREATE TABLE IF NOT EXISTS reputation_overrides (
    override_id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    requested_by VARCHAR(64) NOT NULL,
    reason TEXT NOT NULL,
    delta INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reputation_anomalies (
    anomaly_id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    anomaly_type TEXT NOT NULL,
    confidence NUMERIC(5, 4) NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_reputation_anomalies_user_id
    ON reputation_anomalies (user_id);

CREATE TABLE IF NOT EXISTS reputation_audit_log (
    audit_log_id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    actor_user_id VARCHAR(64) NOT NULL,
    action TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_reputation_audit_log_user_id
    ON reputation_audit_log (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS reputation_outbox (
    outbox_id UUID PRIMARY KEY,
    event_type TEXT NOT NULL,
    partition_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ NULL,
    CONSTRAINT reputation_outbox_status_check CHECK (status IN ('pending', 'published'))
);
CREATE INDEX IF NOT EXISTS idx_reputation_outbox_status_created
    ON reputation_outbox (status, created_at ASC);

CREATE TABLE IF NOT EXISTS reputation_idempotency (
    key TEXT PRIMARY KEY,
    request_hash TEXT NOT NULL,
    response_payload JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_reputation_idempotency_expires_at
    ON reputation_idempotency (expires_at);

CREATE TABLE IF NOT EXISTS reputation_event_dedup (
    event_id TEXT PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_reputation_event_dedup_expires_at
    ON reputation_event_dedup (expires_at);

