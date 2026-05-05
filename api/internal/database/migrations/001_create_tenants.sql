CREATE TABLE IF NOT EXISTS tenants (
    id                              UUID PRIMARY KEY,
    supabase_url_encrypted          TEXT NOT NULL,
    supabase_anon_encrypted         TEXT NOT NULL,
    supabase_service_role_encrypted TEXT NOT NULL,
    supabase_db_url_encrypted       TEXT NOT NULL,
    site_url                        TEXT,
    onboarded_at                    TIMESTAMPTZ,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
