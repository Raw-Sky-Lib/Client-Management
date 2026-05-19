> **Canonical source:** See `Agency-Hub/.claude/FEATURES.md`
> This mirror is kept here for cross-repo context. The authoritative version lives in the agency-hub repo.

# agency-hub — Feature Reference

See the canonical file for full content. Key summary:

## What Exists (Current State)

**No longer exists:**
- `management_token` generate/rotate/validate endpoints — dropped in migration 015
- `deploy_records` table — dropped in migration 018, health status moved to projects
- Project `status` (kanban) — dropped in migration 017

**Portal invite flow (current primary path):**
1. Agency saves Supabase credentials on project → auto-push to portal in background
2. "Push to Portal" button → `POST /projects/{id}/portal/invite` → portal sends magic-link invite
3. Client logs in → portal calls back `POST /clients/{id}/portal-onboarded` → sets `portal_onboarded_at`

**Key endpoints:**
- `PUT /projects/{id}/deploy` — save Supabase credentials + health ping + auto-push to portal
- `POST /projects/{id}/portal/invite` — trigger portal invite (2-min cooldown)
- `POST /clients/{id}/portal-onboarded` — M2M from portal (Bearer PORTAL_ADMIN_SECRET)
- `POST /projects/{id}/deploy/ping` — HTTP GET to `{site_url}/health`, update health_status

**Connection token:** Still in codebase for backward compat. Not the primary path.

See `Agency-Hub/.claude/FEATURES.md` for the full feature inventory.
