CREATE TABLE IF NOT EXISTS form_submissions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    form_name    TEXT NOT NULL,
    data         JSONB NOT NULL DEFAULT '{}',
    is_read      BOOLEAN NOT NULL DEFAULT false,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS form_submissions_form_name_idx ON form_submissions(form_name);
CREATE INDEX IF NOT EXISTS form_submissions_submitted_at_idx ON form_submissions(submitted_at DESC);
