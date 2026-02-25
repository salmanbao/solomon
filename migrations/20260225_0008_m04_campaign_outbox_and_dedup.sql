-- M04-Campaign-Service outbox and consumer dedup support.
-- Safe additive migration: no destructive changes.

CREATE TABLE IF NOT EXISTS campaign_outbox (
    outbox_id UUID PRIMARY KEY,
    event_type VARCHAR(120) NOT NULL,
    partition_key VARCHAR(120) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ NULL,
    CONSTRAINT campaign_outbox_status_check CHECK (status IN ('pending', 'published'))
);

CREATE INDEX IF NOT EXISTS idx_campaign_outbox_status_created
    ON campaign_outbox (status, created_at ASC);

CREATE TABLE IF NOT EXISTS campaign_event_dedup (
    event_id VARCHAR(120) PRIMARY KEY,
    payload_hash VARCHAR(128) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_event_dedup_expires
    ON campaign_event_dedup (expires_at ASC);

