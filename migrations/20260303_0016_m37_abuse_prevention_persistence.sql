-- M37-Abuse-Prevention-Service production persistence hardening.

CREATE TABLE IF NOT EXISTS abuse_lockout_history (
    lockout_id VARCHAR(64) PRIMARY KEY,
    threat_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    reason TEXT NOT NULL DEFAULT '',
    locked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT abuse_lockout_history_status_check CHECK (status IN ('active', 'released'))
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_abuse_lockout_history_active_user
    ON abuse_lockout_history (user_id)
    WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_abuse_lockout_history_user_locked_at
    ON abuse_lockout_history (user_id, locked_at DESC);

CREATE TABLE IF NOT EXISTS abuse_audit_log (
    audit_id VARCHAR(64) PRIMARY KEY,
    actor_id VARCHAR(64) NOT NULL,
    action TEXT NOT NULL,
    target_id VARCHAR(64) NOT NULL,
    justification TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    source_ip TEXT NOT NULL DEFAULT '',
    correlation_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_abuse_audit_log_occurred_at
    ON abuse_audit_log (occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_abuse_audit_log_target_id
    ON abuse_audit_log (target_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS abuse_idempotency (
    key TEXT PRIMARY KEY,
    request_hash TEXT NOT NULL,
    response_body BYTEA NOT NULL DEFAULT ''::bytea,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_abuse_idempotency_expires_at
    ON abuse_idempotency (expires_at);
