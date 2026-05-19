> **Canonical source:** See `Agency-Hub/.claude/PROJECT-STRUCTURE.md`

# agency-hub — Project Structure (Summary)

## Backend Internal Packages

```
internal/
├── auth/          Login, logout, refresh, CSRF, profile, invite flow, password reset
├── team/          Team CRUD, invite, deactivate/reactivate, RBAC roles
├── client/        Client CRUD, connection token (legacy), portal-onboarded callback
├── project/       Project CRUD, deploy credentials, health ping, portal invite
│   └── portal.go  PortalCaller — M2M calls to portal backend (PORTAL_ADMIN_SECRET)
├── dashboard/     GET /dashboard aggregate stats
├── audit/         Internal-only audit logging service (no HTTP handler)
├── mailer/        Pluggable: Resend or Brevo (MAILER_PROVIDER env)
├── config/        Typed config struct from env vars
├── database/      database/sql connection + migration runner
├── middleware/    auth, rbac, csrf, ratelimit, security headers, logger
└── utils/         crypto (GenerateToken, HashToken), response, errors
```

**No `deploy/` package** — deploy fields live directly on `projects` table (migration 018).

## Current DB Schema (condensed)

```
clients:   id, name, email, status, connection_token_*, github_repo_url,
           portal_invite_sent_at, portal_onboarded_at, created_by, created_at
           (management_token dropped, supabase_* moved to projects)

projects:  id, client_id, name, supabase_ref, supabase_url, supabase_anon_key,
           supabase_service_role_key, supabase_db_url, site_url, github_repo_url,
           health_status, last_checked_at, portal_registered_at, last_invited_at
           (no status column — kanban dropped in migration 017)
```

See `Agency-Hub/.claude/PROJECT-STRUCTURE.md` for the full file tree.
