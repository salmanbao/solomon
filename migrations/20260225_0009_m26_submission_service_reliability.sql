-- M26-Submission-Service reliability + schema parity.
-- Safe additive migration: no destructive drops of owned tables.

ALTER TABLE submissions
    ADD COLUMN IF NOT EXISTS post_id TEXT,
    ADD COLUMN IF NOT EXISTS creator_platform_handle TEXT,
    ADD COLUMN IF NOT EXISTS verification_start TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS views_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS locked_views INTEGER NULL,
    ADD COLUMN IF NOT EXISTS locked_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS last_view_sync TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS cpv_rate DECIMAL(10, 4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS gross_amount DECIMAL(12, 4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS platform_fee DECIMAL(12, 4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS net_amount DECIMAL(12, 4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_status_check;
ALTER TABLE submissions
    ADD CONSTRAINT submissions_status_check
    CHECK (status IN (
        'pending',
        'approved',
        'rejected',
        'flagged',
        'verification_period',
        'view_locked',
        'reward_eligible',
        'paid',
        'disputed',
        'cancelled'
    ));

CREATE INDEX IF NOT EXISTS idx_submissions_platform_post_id
    ON submissions (platform, post_id);
CREATE INDEX IF NOT EXISTS idx_submissions_verification_window_end
    ON submissions (verification_window_end)
    WHERE verification_window_end IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_submissions_locked_views
    ON submissions (locked_views DESC)
    WHERE locked_views IS NOT NULL;

ALTER TABLE submissions_audit
    ADD COLUMN IF NOT EXISTS actor_role TEXT NULL,
    ADD COLUMN IF NOT EXISTS ip_address TEXT NULL,
    ADD COLUMN IF NOT EXISTS user_agent TEXT NULL;

ALTER TABLE submission_flags
    ADD COLUMN IF NOT EXISTS is_resolved BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ NULL;

ALTER TABLE view_snapshots
    ADD COLUMN IF NOT EXISTS platform_metrics JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE bulk_submission_operations
    ADD COLUMN IF NOT EXISTS performed_by_user_id UUID NULL,
    ADD COLUMN IF NOT EXISTS reason_code TEXT NULL,
    ADD COLUMN IF NOT EXISTS reason_notes TEXT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'bulk_submission_operations_type_check'
    ) THEN
        ALTER TABLE bulk_submission_operations
            ADD CONSTRAINT bulk_submission_operations_type_check
            CHECK (operation_type IN ('bulk_approve', 'bulk_reject'));
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS submission_idempotency (
    key TEXT PRIMARY KEY,
    request_hash TEXT NOT NULL,
    response_payload BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS submission_outbox (
    outbox_id UUID PRIMARY KEY,
    event_type TEXT NOT NULL,
    partition_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_submission_outbox_status_created
    ON submission_outbox (status, created_at ASC);

CREATE TABLE IF NOT EXISTS submission_event_dedup (
    event_id UUID PRIMARY KEY,
    payload_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_submission_event_dedup_expires_at
    ON submission_event_dedup (expires_at ASC);
