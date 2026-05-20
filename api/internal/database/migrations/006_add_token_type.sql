-- Distinguish invite tokens (onboarding confirm) from login tokens (magic link, password reset).
-- Existing rows default to 'invite'; any in-flight login tokens are 1-hour-lived and will
-- have expired before this migration runs in practice.
ALTER TABLE email_confirmations
    ADD COLUMN IF NOT EXISTS token_type TEXT NOT NULL DEFAULT 'invite';

CREATE INDEX IF NOT EXISTS email_confirmations_email_type_idx
    ON email_confirmations(email, token_type);
