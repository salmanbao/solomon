-- M04-Campaign-Service additive columns for canonical spec parity.
-- Safe additive migration: no destructive changes.

ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS required_tags TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS optional_hashtags TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS usage_guidelines TEXT,
    ADD COLUMN IF NOT EXISTS dos_and_donts TEXT,
    ADD COLUMN IF NOT EXISTS campaign_type VARCHAR(20) NOT NULL DEFAULT 'ugc_creation',
    ADD COLUMN IF NOT EXISTS deadline TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS target_submissions INTEGER NULL,
    ADD COLUMN IF NOT EXISTS banner_image_url TEXT NULL,
    ADD COLUMN IF NOT EXISTS external_url TEXT NULL,
    ADD COLUMN IF NOT EXISTS budget_reserved DECIMAL(12, 2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS submission_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS approved_submission_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_views BIGINT NOT NULL DEFAULT 0;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'campaigns_campaign_type_check'
    ) THEN
        ALTER TABLE campaigns
            ADD CONSTRAINT campaigns_campaign_type_check
            CHECK (campaign_type IN ('ugc_creation', 'ugc_distribution', 'hybrid'));
    END IF;
END $$;

