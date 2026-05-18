-- ─── 004_add_tenant_projects.sql ─────────────────────────────────────────────
-- Moves Supabase credentials from the tenants table (one-per-tenant) to a new
-- tenant_projects table (many-per-tenant), enabling one client to manage
-- multiple projects from a single portal login.
--
-- Also scopes email_confirmations to a project, since a connection token
-- from agency-hub is issued per-project, not per-client.

-- ── 1. tenant_projects ────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS tenant_projects (
    id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    agency_project_id               UUID NOT NULL UNIQUE,
    name                            TEXT NOT NULL,
    site_url                        TEXT,
    supabase_url_encrypted          TEXT NOT NULL,
    supabase_anon_encrypted         TEXT NOT NULL,
    supabase_service_role_encrypted TEXT NOT NULL,
    supabase_db_url_encrypted       TEXT,
    registered_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS tenant_projects_tenant_id_idx ON tenant_projects(tenant_id);

-- ── 2. Migrate any existing tenant credentials ────────────────────────────────
-- agency_project_id is unknowable from portal data alone, so a placeholder UUID
-- is generated. These rows should be re-registered via agency-hub after deploy.

INSERT INTO tenant_projects (
    tenant_id,
    agency_project_id,
    name,
    site_url,
    supabase_url_encrypted,
    supabase_anon_encrypted,
    supabase_service_role_encrypted,
    supabase_db_url_encrypted,
    registered_at,
    created_at
)
SELECT
    id,
    gen_random_uuid(),
    'Default',
    site_url,
    supabase_url_encrypted,
    supabase_anon_encrypted,
    supabase_service_role_encrypted,
    supabase_db_url_encrypted,
    COALESCE(onboarded_at, NOW()),
    created_at
FROM tenants
WHERE supabase_url_encrypted IS NOT NULL
  AND supabase_url_encrypted <> ''
ON CONFLICT DO NOTHING;

-- ── 3. Drop credential columns from tenants ───────────────────────────────────
-- tenants now stores identity only (id = client_id, onboarded_at, created_at).

ALTER TABLE tenants
    DROP COLUMN IF EXISTS supabase_url_encrypted,
    DROP COLUMN IF EXISTS supabase_anon_encrypted,
    DROP COLUMN IF EXISTS supabase_service_role_encrypted,
    DROP COLUMN IF EXISTS supabase_db_url_encrypted,
    DROP COLUMN IF EXISTS site_url;

-- ── 4. Scope email confirmations to a project ─────────────────────────────────
-- Connection tokens are issued per-project in agency-hub. The confirmation
-- row needs to carry the project so Stage 2 onboarding can create the right
-- tenant_projects row on /confirm.

ALTER TABLE email_confirmations
    ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES tenant_projects(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS email_confirmations_project_id_idx ON email_confirmations(project_id);
