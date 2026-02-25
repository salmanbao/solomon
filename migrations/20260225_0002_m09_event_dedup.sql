-- M09 distribution event dedupe table used by worker consumers.

CREATE TABLE IF NOT EXISTS content_marketplace_event_dedup (
    event_id TEXT PRIMARY KEY,
    payload_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_content_marketplace_event_dedup_expires_at
    ON content_marketplace_event_dedup (expires_at);
