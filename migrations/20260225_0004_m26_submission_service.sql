-- M26-Submission-Service initial schema.
-- Ownership: submission-service (single-writer logical ownership).

CREATE TABLE IF NOT EXISTS submissions (
    submission_id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL,
    creator_id UUID NOT NULL,
    platform VARCHAR(32) NOT NULL,
    post_url TEXT NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'pending',
    approved_at TIMESTAMPTZ NULL,
    approved_by_user_id UUID NULL,
    approval_reason TEXT NULL,
    rejected_at TIMESTAMPTZ NULL,
    rejection_reason TEXT NULL,
    rejection_notes TEXT NULL,
    reported_count INTEGER NOT NULL DEFAULT 0,
    verification_window_end TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT submissions_status_check CHECK (status IN ('pending', 'approved', 'rejected', 'flagged', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_submissions_campaign_id ON submissions (campaign_id);
CREATE INDEX IF NOT EXISTS idx_submissions_creator_id ON submissions (creator_id);
CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions (status);
CREATE INDEX IF NOT EXISTS idx_submissions_created_at ON submissions (created_at DESC);

CREATE TABLE IF NOT EXISTS submissions_audit (
    audit_id UUID PRIMARY KEY,
    submission_id UUID NOT NULL REFERENCES submissions (submission_id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    old_status VARCHAR(24) NULL,
    new_status VARCHAR(24) NULL,
    actor_id UUID NULL,
    reason_code TEXT NULL,
    reason_notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_submissions_audit_submission_id ON submissions_audit (submission_id, created_at DESC);

CREATE TABLE IF NOT EXISTS submission_flags (
    flag_id UUID PRIMARY KEY,
    submission_id UUID NOT NULL REFERENCES submissions (submission_id) ON DELETE CASCADE,
    flag_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'low',
    details JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_submission_flags_submission_id ON submission_flags (submission_id, created_at DESC);

CREATE TABLE IF NOT EXISTS submission_reports (
    report_id UUID PRIMARY KEY,
    submission_id UUID NOT NULL REFERENCES submissions (submission_id) ON DELETE CASCADE,
    reported_by_user_id UUID NULL,
    reason VARCHAR(50) NOT NULL,
    description TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_submission_reports_submission_id ON submission_reports (submission_id, created_at DESC);

CREATE TABLE IF NOT EXISTS bulk_submission_operations (
    operation_id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL,
    operation_type VARCHAR(20) NOT NULL,
    submission_ids UUID[] NOT NULL DEFAULT '{}',
    succeeded_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    reason TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bulk_submission_operations_campaign_id
    ON bulk_submission_operations (campaign_id, created_at DESC);

CREATE TABLE IF NOT EXISTS view_snapshots (
    snapshot_id UUID PRIMARY KEY,
    submission_id UUID NOT NULL REFERENCES submissions (submission_id) ON DELETE CASCADE,
    views_count INTEGER NOT NULL DEFAULT 0,
    engagement_estimate INTEGER NOT NULL DEFAULT 0,
    platform_metrics JSONB NOT NULL DEFAULT '{}',
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_anomaly BOOLEAN NOT NULL DEFAULT FALSE,
    anomaly_reason TEXT NULL
);

CREATE INDEX IF NOT EXISTS idx_view_snapshots_submission_id ON view_snapshots (submission_id, synced_at DESC);

