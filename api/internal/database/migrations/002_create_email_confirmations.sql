CREATE TABLE IF NOT EXISTS email_confirmations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS email_confirmations_token_hash_idx ON email_confirmations(token_hash);
CREATE INDEX IF NOT EXISTS email_confirmations_tenant_id_idx ON email_confirmations(tenant_id);
