> **This file mirrors the agency-hub repo's `.claude/CLAUDE.md` for cross-repo context.**
> **Canonical source:** `/Users/dagi/Documents/Github/Matt x Dagim/Agency-Hub/.claude/CLAUDE.md`

# CLAUDE.md — agency-hub
> Internal agency management tool. Dagim + Matt + team only.
> RBAC: super_admin · admin · moderator

---

## What This App Is

agency-hub is the agency's internal command centre. It manages client records, projects, deploy credentials, GitHub repo setup, team members, and the portal invite lifecycle. It has no public-facing component — only agency team members have access.

The app has two outbound integrations:
- **Portal** (`PORTAL_URL` + `PORTAL_ADMIN_SECRET`): pushes project Supabase credentials to the client portal and sends the client's magic-link invite
- **GitHub** (`GITHUB_*` env vars): creates and initialises repositories via the GitHub API on-demand

---

## Stack

**Backend (`api/`)**
- Go 1.24+ · Chi v5
- PostgreSQL via Supabase (agency's own project, `database/sql`)
- JWT: 15-min access token, 7-day refresh, HTTP-only cookies
- Upstash Redis: rate limiting (sliding window) + session tracking
- Resend / Brevo (pluggable mailer): transactional email — team invites, password setup
- Swagger (swag): API docs at `/swagger/index.html`
- slog: structured logging (`pkg/logger`)
- go-playground/validator v10: request body validation
- Feature-based folder: `internal/<feature>/{model,repository,service,handler,routes}.go`

**Frontend (`web/`)**
- React 19 · TypeScript (strict) · Vite
- Tailwind CSS v4 · shadcn/ui · Radix UI
- TanStack Query v5 · React Router v7
- React Hook Form + Zod · Framer Motion · Sonner · Lucide React
- Feature-based folder: `src/features/<feature>/{pages,components,hooks,services,types}/`

---

## RBAC — Roles & Access

```
super_admin   Full access. Can manage roles, delete records, view all data.
admin         Full operational access. Create clients, manage projects, push portal invites.
moderator     Read-only across all areas. No write operations.
```

**JWT claims:**
```go
type Claims struct {
    UserID   string `json:"user_id"`
    Email    string `json:"email"`
    Role     string `json:"role"`     // "super_admin" | "admin" | "moderator"
    IsActive bool   `json:"is_active"`
    jwt.RegisteredClaims
}
```

---

## DB Schema (Agency Supabase — current, post all migrations)

```sql
users(id UUID PK, email TEXT UNIQUE, name TEXT, role TEXT, password_hash TEXT,
      is_active BOOL DEFAULT false, created_at TIMESTAMPTZ)

role_requests(id, requested_by → users, target_email, requested_role, status, reviewed_by, created_at)

user_invites(id, user_id → users, token_hash TEXT UNIQUE, expires_at, accepted_at, created_at)

password_reset_tokens(id, user_id → users, token_hash TEXT UNIQUE, expires_at, used_at, created_at)

clients(
  id UUID PK,
  name TEXT, business_name TEXT, email TEXT UNIQUE, phone TEXT,
  plan_tier TEXT CHECK ('starter','growth','scale'),
  status TEXT CHECK ('active','paused','churned') DEFAULT 'active',
  notes TEXT,
  -- Connection token (legacy portal flow)
  connection_token_hash TEXT UNIQUE, connection_token_expires_at TIMESTAMPTZ,
  connection_token_used_at TIMESTAMPTZ,
  -- GitHub
  github_repo_url TEXT,
  -- Portal invite lifecycle
  portal_invite_sent_at TIMESTAMPTZ,
  portal_onboarded_at   TIMESTAMPTZ,
  created_by → users, created_at TIMESTAMPTZ
)
-- Note: management_token columns dropped in migration 015.

projects(
  id UUID PK, client_id → clients, name TEXT, description TEXT,
  linear_project_url TEXT, notion_page_url TEXT,
  estimated_delivery DATE, actual_delivery DATE, assigned_to → users,
  -- Supabase credentials (service_role_key never returned in API responses)
  supabase_ref TEXT, supabase_url TEXT, supabase_anon_key TEXT,
  supabase_service_role_key TEXT, supabase_db_url TEXT,
  -- Deploy / health
  site_url TEXT, github_repo_url TEXT,
  health_status TEXT DEFAULT 'unknown',
  last_checked_at TIMESTAMPTZ,
  -- Portal registration state
  portal_registered_at TIMESTAMPTZ, last_invited_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ
)
-- Note: status column dropped in migration 017. No kanban status on projects.

audit_logs(id, actor_id, action TEXT, entity_type TEXT, entity_id TEXT,
           metadata JSONB, ip TEXT, created_at TIMESTAMPTZ)

claude_usage(id, client_id → clients, year_month TEXT, requests INT,
             tokens_input INT, tokens_output INT, last_used_at, created_at)
```

---

## Client Lifecycle & Portal Integration

### Supabase credentials
Admin saves credentials on project detail page (`PUT /projects/{id}/deploy`). On save, the backend auto-pushes credentials to the portal in a background goroutine.

### Portal invite ("Push to Portal")
`POST /projects/{id}/portal/invite`:
1. Calls `POST /api/admin/register-client` on portal backend (Bearer `PORTAL_ADMIN_SECRET`)
2. Portal registers tenant + project credentials + sends magic-link invite email
3. Updates `portal_registered_at` and `last_invited_at` on the project
4. Updates `portal_invite_sent_at` on the parent client
5. 2-minute cooldown to prevent duplicate sends

### Portal onboarded callback
When the client accepts their invite and logs in, the portal calls back:
```
POST /api/clients/{id}/portal-onboarded
Authorization: Bearer <PORTAL_ADMIN_SECRET>
```
Sets `portal_onboarded_at` on the client.

### Portal invite status (shown on ClientDetailPage)
- `portal_onboarded_at` set → **ACTIVE** (green)
- `portal_invite_sent_at` set → **INVITE SENT** (yellow)
- Neither → **NOT INVITED** (grey)

---

## Token Architecture

**`connection_token`** (legacy client onboarding):
- Still in codebase for backward compat
- 32-byte hex, SHA-256 hashed, 7-day expiry, one-time use
- The primary onboarding path is now the portal invite flow above

**`invite_token`** (team member onboarding):
- 32-byte hex, SHA-256 hashed, stored in `user_invites`
- Sent in setup-password URL
- `POST /api/auth/invite/setup-password` validates + activates the account

**Server-to-server auth** (`PORTAL_ADMIN_SECRET`):
- Shared secret between agency-hub and portal backend
- Used in `Authorization: Bearer <PORTAL_ADMIN_SECRET>` on machine-to-machine endpoints
- Not stored in DB — env var only

---

## Backend Conventions

```
internal/<feature>/
├── model.go        → DB row structs, request/response types
├── repository.go   → All DB queries (parameterised only), NewRepository(db)
├── service.go      → Business logic; calls repo, mailer, audit
├── handler.go      → HTTP handlers with Swagger annotations
└── routes.go       → Chi route registration + middleware wiring
```

---

## Frontend Conventions

```
src/features/<feature>/
├── pages/<PageName>.tsx          → route-level component; uses hooks only
├── components/<Name>.tsx         → feature-specific UI
├── hooks/use-<feature>.ts        → TanStack Query wrappers
├── services/<feature>.service.ts → axios calls
└── types/index.ts                → TypeScript interfaces
```

**Mutation hook pattern:**
```typescript
export function useDeleteClient() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => clientsService.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: CLIENTS_KEY })
      qc.invalidateQueries({ queryKey: ['dashboard', 'stats'] })
    },
    onError: (err: unknown) => toast.error(parseApiError(err) || 'Failed to delete client'),
  })
}
```

---

## Security Rules

- CSRF token required on ALL POST/PUT/PATCH/DELETE routes
- Input validation via `validator.New().Struct()` on every request body
- Rate limiting: 5 req/min on auth routes, 30 req/min on authenticated routes
- Account lockout: 5 failed logins → 15-min lockout (Redis-backed)
- Never store plaintext tokens — SHA-256 hash before writing to DB
- `DB_SSLMODE=require` enforced in production
- `PORTAL_ADMIN_SECRET` is the only secret shared between agency-hub and portal

---

## Environment Variables

```env
APP_ENV=development
APP_PORT=8080
DB_HOST=db.<ref>.supabase.co
DB_PORT=5432
DB_NAME=postgres
DB_USER=postgres
DB_PASSWORD=
DB_SSLMODE=require
JWT_ACCESS_SECRET=
JWT_REFRESH_SECRET=
JWT_ISSUER=agency-hub
REDIS_URL=
REDIS_TOKEN=
MAILER_PROVIDER=resend            # or "brevo"
RESEND_API_KEY=
MAIL_FROM=noreply@youragency.com
PORTAL_URL=https://portal.yourdomain.com
PORTAL_ADMIN_SECRET=<shared secret>
GITHUB_TOKEN=
GITHUB_ORG=
```

---

## Do Not

**Backend:**
- Do not use string interpolation in SQL — parameterised only
- Do not return stack traces or Go error strings to HTTP clients
- Do not skip Swagger annotations on any handler
- Do not skip audit logging on data-modifying actions
- Do not store plaintext tokens
- Do not expose `supabase_service_role_key` in any API response
- Do not filter projects by `status` — column does not exist (migration 017)

**Frontend:**
- Do not use `useEffect` for data fetching — TanStack Query only
- Do not create new UI primitives — use shadcn/ui
- Do not use `any`
- Do not deviate from the established visual patterns
