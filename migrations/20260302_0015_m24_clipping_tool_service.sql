-- M24-Clipping-Tool-Service canonical ownership tables.

CREATE TABLE IF NOT EXISTS user_projects (
    project_id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    title TEXT NOT NULL,
    description TEXT NULL,
    thumbnail_url TEXT NULL,
    state VARCHAR(32) NOT NULL DEFAULT 'draft',
    source_url TEXT NULL,
    source_type VARCHAR(24) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT user_projects_state_check CHECK (state IN ('draft', 'in_progress', 'completed'))
);
CREATE INDEX IF NOT EXISTS idx_user_projects_user_updated
    ON user_projects (user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS project_timelines (
    timeline_id UUID PRIMARY KEY,
    project_id UUID NOT NULL UNIQUE,
    clips JSONB NOT NULL DEFAULT '[]'::jsonb,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    fps INTEGER NOT NULL DEFAULT 30,
    width INTEGER NOT NULL DEFAULT 1080,
    height INTEGER NOT NULL DEFAULT 1920,
    export_settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT project_timelines_fps_check CHECK (fps IN (30, 60)),
    CONSTRAINT project_timelines_duration_check CHECK (duration_ms >= 0)
);

CREATE TABLE IF NOT EXISTS project_versions (
    version_id UUID PRIMARY KEY,
    project_id UUID NOT NULL,
    version_number INTEGER NOT NULL,
    timeline_state JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id UUID NOT NULL,
    change_summary TEXT NULL,
    CONSTRAINT project_versions_number_check CHECK (version_number > 0)
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_project_versions_project_number
    ON project_versions (project_id, version_number);

CREATE TABLE IF NOT EXISTS export_jobs (
    export_id UUID PRIMARY KEY,
    project_id UUID NOT NULL,
    user_id UUID NOT NULL,
    format VARCHAR(24) NOT NULL DEFAULT 'mp4',
    resolution VARCHAR(32) NOT NULL DEFAULT '1080x1920',
    fps INTEGER NOT NULL DEFAULT 30,
    bitrate VARCHAR(16) NOT NULL DEFAULT '10m',
    status VARCHAR(32) NOT NULL DEFAULT 'queued',
    progress_percent INTEGER NOT NULL DEFAULT 0,
    provider_job_id TEXT NULL,
    output_url TEXT NULL,
    error_message TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ NULL,
    CONSTRAINT export_jobs_format_check CHECK (format IN ('mp4')),
    CONSTRAINT export_jobs_fps_check CHECK (fps IN (30, 60)),
    CONSTRAINT export_jobs_status_check CHECK (status IN ('queued', 'processing', 'completed', 'failed')),
    CONSTRAINT export_jobs_progress_check CHECK (progress_percent >= 0 AND progress_percent <= 100)
);
CREATE INDEX IF NOT EXISTS idx_export_jobs_project_created
    ON export_jobs (project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_export_jobs_user_status
    ON export_jobs (user_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS export_campaign_submissions (
    submission_id UUID PRIMARY KEY,
    export_id UUID NOT NULL,
    campaign_id UUID NOT NULL,
    user_id UUID NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'submitted',
    post_url TEXT NULL,
    platform VARCHAR(32) NOT NULL DEFAULT 'tiktok',
    submitted_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT export_campaign_submissions_status_check CHECK (status IN ('draft', 'submitted', 'approved', 'rejected'))
);
CREATE INDEX IF NOT EXISTS idx_export_campaign_submissions_campaign
    ON export_campaign_submissions (campaign_id, user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS editor_usage_analytics (
    usage_id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    project_id UUID NULL,
    session_start TIMESTAMPTZ NOT NULL,
    session_end TIMESTAMPTZ NULL,
    duration_seconds INTEGER NOT NULL DEFAULT 0,
    edits_count INTEGER NOT NULL DEFAULT 0,
    exports_count INTEGER NOT NULL DEFAULT 0,
    exports_success INTEGER NOT NULL DEFAULT 0,
    device TEXT NULL,
    browser_info TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_editor_usage_analytics_user_session
    ON editor_usage_analytics (user_id, session_start DESC);

