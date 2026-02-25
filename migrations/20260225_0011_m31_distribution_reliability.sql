-- M31-Distribution-Service reliability additions.
-- Adds outbox for asynchronous event publication and uniqueness guard for claim ingestion.

CREATE TABLE IF NOT EXISTS distribution_outbox (
    outbox_id UUID PRIMARY KEY,
    event_type TEXT NOT NULL,
    partition_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_distribution_outbox_status_created_at
    ON distribution_outbox (status, created_at);

CREATE INDEX IF NOT EXISTS idx_distribution_items_scheduled_for_utc
    ON distribution_items (scheduled_for_utc);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'distribution_items_unique_claim'
    ) THEN
        ALTER TABLE distribution_items
            ADD CONSTRAINT distribution_items_unique_claim
                UNIQUE (influencer_id, clip_id, campaign_id);
    END IF;
END $$;
