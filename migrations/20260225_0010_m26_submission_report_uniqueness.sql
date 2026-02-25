-- M26-Submission-Service: prevent duplicate reports from same reporter on same submission.
-- Safe additive index.

CREATE UNIQUE INDEX IF NOT EXISTS idx_submission_reports_unique_reporter
    ON submission_reports (submission_id, reported_by_user_id)
    WHERE reported_by_user_id IS NOT NULL;
