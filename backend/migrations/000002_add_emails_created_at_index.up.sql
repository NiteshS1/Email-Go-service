-- Index on created_at for time-ordered listing and filtering
CREATE INDEX IF NOT EXISTS idx_emails_created_at ON emails (created_at DESC);
