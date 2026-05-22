ALTER TABLE tenant_projects ADD COLUMN IF NOT EXISTS revalidate_secret_encrypted TEXT;
