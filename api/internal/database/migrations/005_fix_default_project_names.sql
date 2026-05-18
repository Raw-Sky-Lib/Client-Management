-- ─── 005_fix_default_project_names.sql ───────────────────────────────────────
-- Migration 004 inserted existing tenant credentials with the placeholder name
-- 'Default'. This migration replaces it with a name derived from site_url
-- (e.g. https://dagim.com → dagim.com) so users see something meaningful.

UPDATE tenant_projects
SET name = CASE
    WHEN site_url IS NOT NULL AND site_url <> ''
    THEN regexp_replace(
             regexp_replace(site_url, '^https?://(www\.)?', ''),
             '/.*$', ''
         )
    ELSE 'My Project'
END
WHERE name = 'Default';
