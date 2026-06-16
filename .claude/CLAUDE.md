# CLAUDE.md — client-portal
> Multi-tenant CMS dashboard. One deployment serves all agency clients.
> Each client is isolated in their own Supabase project.
> Stack: Go 1.24+ backend · React 19 + Vite frontend · Supabase

---

## Task workflow (finalization phase)

Active work is tracked in `.claude/tasks.md` (CP-1 → CP-18). The user reviews and tests every task before the next one starts. Follow this loop strictly:

1. Start one task at a time. Announce which task ID you're starting.
2. Do the work. Don't bundle multiple tasks.
3. **Always end with a "How to test" section** — concrete repro steps the user can run, expected outcome, what to look for. No vague "try it out" — give exact URLs, buttons, commands, and acceptance signals.
4. Ask the user to review. Wait for explicit go-ahead before starting the next task.
5. When the user confirms it passes, mark the task `[✓]` in `tasks.md` with the date, then propose the next one.

The same loop applies to the Agency-Hub tasks (AH-1 → AH-13) — see that repo's `tasks.md`.

### Template / test-tenant sync

`/Users/dagi/Documents/Github/Matt x Dagim/client-dagim-digital-agency/` is a live client site built from `format-studio-client-site-template` a while ago. The user runs it as the test tenant against the portal. **It does NOT pull updates automatically.** Whenever a task modifies the template, mirror the equivalent change in this client repo in the same step (copy new files verbatim, port edits to existing files with care — beware of brand-specific differences in `globals.css`, `layout.tsx`, fonts, etc.). Confirm the sync in the same "How to test" section so the user can test against a real Supabase-backed site.

---

## What This App Is

client-portal is the CMS dashboard that agency clients use to manage their website content — pages, blog posts, media, form submissions, settings, and an AI writing assistant. It is multi-tenant: one hosted instance serves every client, each reading/writing their own Supabase project.

**The portal backend is a thin authority layer.** It:
- Registers tenants and stores their encrypted Supabase credentials (pushed by agency-hub)
- Sends onboarding invite emails and handles the magic-link auth flow
- Issues the portal JWT and validates sessions
- Proxies Claude API calls (API key never reaches the browser)
- Triggers ISR revalidation on the client's live site after content updates
- Runs DB migrations on the client's Supabase on first registration

The React frontend does all CMS reads and writes directly to the client's Supabase via the JS client, using the anon key and RLS.

---

## Stack

**Backend (`api/`)**
- Go 1.24+ · Chi v5
- PostgreSQL via Supabase (portal's own project — tenant registry only, not CMS)
- pgx/v5 connection pool
- JWT: 15-min access, 7-day refresh, HTTP-only cookies
- Upstash Redis: rate limiting (sliding window, auth + Claude)
- Resend / Brevo (pluggable via `MAILER_PROVIDER` env var): onboarding invite email
- Anthropic Go SDK: Claude Haiku — content assistant proxy
- slog: structured logging · go-playground/validator v10

**Frontend (`web/`)**
- React 19 · TypeScript (strict) · Vite · port 5174
- Tailwind CSS v4 · shadcn/ui · Radix UI
- TanStack Query v5 · React Router v7
- React Hook Form + Zod · Framer Motion · Sonner · Lucide React
- Supabase JS client (per-tenant, initialised per active project)
- Tiptap: rich text editor for blog posts

---

## Two Connections — Never Mix Them

```
Connection A — Portal backend API  (axios at /api/*)
  Used for: auth (login, magic link, JWT), projects list, Claude proxy, ISR trigger
  Auth: portal JWT in HTTP-only cookie
  NOT used for: reading or writing CMS content

Connection B — Client's Supabase project  (Supabase JS client)
  Used for: all CMS content — pages, posts, media, settings, form submissions
  Auth: anon key (RLS enforced) or service role key (backend only, never returned to browser)
  NOT used for: portal auth, session management, or anything the agency-hub owns
```

When writing any feature, be explicit about which connection it uses.

---

## Auth Flow (current — no connection_token)

The connection_token / `/connect` page flow has been **replaced**. The current flow is:

```
1. Agency creates client + project in agency-hub
2. Agency saves Supabase credentials → triggers auto-push to portal in background
   OR agency clicks "Push to Portal" button explicitly
3. Portal backend (POST /api/admin/register-client from agency-hub):
   a. UPSERTs tenant row in portal DB
   b. Validates Supabase credentials
   c. Runs DB migrations on client's Supabase
   d. Creates default media storage bucket
   e. Encrypts + stores project credentials in tenant_projects
   f. Sends magic-link invite email to client (via Supabase Auth)
4. Client receives email → clicks magic link → Supabase redirects to /auth/callback
5. AuthCallbackPage extracts Supabase access_token → POST /api/auth/exchange
6. Portal backend verifies Supabase token, looks up tenant by email, issues portal JWT
7. JWT set in HTTP-only cookie → redirect to /welcome
8. Portal calls back to agency-hub: POST /api/clients/{id}/portal-onboarded (bearer PORTAL_ADMIN_SECRET)
9. Client navigates to /dashboard → full CMS access
```

**Returning logins:** magic link (POST /api/auth/magic-link → Supabase magic link) OR password (POST /api/auth/login). Both result in a portal JWT after exchange or direct credential check.

---

## Portal JWT Claims (lean — no Supabase creds embedded)

```go
type PortalClaims struct {
    UserID   string `json:"user_id"`
    TenantID string `json:"tenant_id"`  // = client_id from agency-hub
    Email    string `json:"email"`
    jwt.RegisteredClaims
}
```

Supabase credentials are **NOT** in the JWT. They are fetched at runtime via `GET /api/projects` (decrypted server-side from `tenant_projects` table).

---

## Multi-Project Architecture

A client can have more than one project (e.g., main site + separate landing page). The `SupabaseProvider` handles this:

```typescript
// src/contexts/supabase-context.tsx
// Fetches all projects for the authenticated tenant on mount.
// Exposes: supabase (client for active project), activeProject, projects, setActiveProjectId
const { data } = useQuery({ queryKey: ['projects'], queryFn: () => api.get('/api/projects') })
```

- `GET /api/projects` returns all `tenant_projects` rows for the tenant (decrypted `supabase_url` + `supabase_anon_key` only — service role never sent to browser)
- `setActiveProjectId` switches the active Supabase client and clears all cached CMS queries
- All CMS hooks call `useTenantSupabase()` — never initialise a Supabase client directly
- If no projects exist yet: `SupabaseProvider` shows a "no project set up" screen and polls every 10s
- `/welcome` route uses `AuthOnlyRoute` (not `ProtectedRoute`) so it works before any project is registered

---

## Portal DB Schema

```sql
-- Tenant identity (one row per client)
tenants(
  id UUID PK,           -- = client_id from agency-hub
  onboarded_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ
)

-- Per-project credentials (one row per agency project pushed to portal)
tenant_projects(
  id UUID PK,
  tenant_id → tenants,
  agency_project_id TEXT,     -- project ID from agency-hub
  name TEXT,
  supabase_url_encrypted TEXT,
  supabase_anon_encrypted TEXT,
  supabase_service_role_encrypted TEXT,
  supabase_db_url_encrypted TEXT,
  site_url TEXT,
  created_at TIMESTAMPTZ
)

-- Email → tenant mapping for future logins
tenant_users(
  id UUID PK,
  tenant_id → tenants,
  email TEXT,
  created_at TIMESTAMPTZ,
  UNIQUE(tenant_id, email)
)

-- Email confirmation tokens (for invite/reset flows)
email_confirmations(
  id UUID PK,
  tenant_id → tenants,
  project_id UUID,       -- which tenant_project this is for
  email TEXT,
  token_hash TEXT UNIQUE,
  expires_at TIMESTAMPTZ,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ
)
```

---

## Credential Encryption

Client Supabase credentials are stored AES-256-GCM encrypted in the portal DB.
- Key: derived from `JWT_SECRET` via HKDF-SHA256 to 32 bytes
- Encrypt/decrypt: `internal/utils/crypto.go` — `EncryptString` / `DecryptString`
- Never log credentials. Never return service role key in any API response.

---

## Backend Package Map

```
internal/
├── onboarding/    RegisterClient (from agency-hub), SendInvite, ResendInvite, Confirm, DeregisterClient
├── auth/          Login (password), MagicLink, LoginVerify, Exchange, Refresh, Logout, Profile,
│                  SetPassword, ResetPasswordRequest/Verify/Confirm, CSRF
├── portalproject/ GET /api/projects → returns decrypted project list for SupabaseProvider
├── tenant/        ResolveTenant middleware — decrypts creds from DB into request context
├── claude/        Rate limit, budget check, Anthropic call, usage recording
├── revalidate/    POST /api/revalidate → fires ISR on client's live site (fire-and-forget)
├── media/         Signed URL generation for Supabase Storage
├── startup/       ValidateManagementToken on boot (os.Exit(1) if fails)
├── mailer/        Pluggable: Resend or Brevo (set via MAILER_PROVIDER env)
├── middleware/     JWT auth, CSRF, rate limit, CORS, security headers, logger
├── config/        LoadConfig() — all env vars
├── database/      pgxpool connection + MigrateClientDB (runs migrations on client Supabase)
└── utils/         crypto.go, errors.go, response.go, supabase.go
```

---

## Admin Endpoints (machine-to-machine, no JWT/CSRF)

Called by agency-hub backend only. All authenticated by `Authorization: Bearer <PORTAL_ADMIN_SECRET>`.

```
POST /api/admin/register-client    → RegisterClientRequest — upsert tenant + project creds + send invite
POST /api/admin/send-invite        → {client_id, email} — send/resend magic link invite
POST /api/admin/resend-invite      → {client_id, email} — explicit resend
DELETE /api/admin/deregister-client/{client_id} → remove tenant from portal (client deleted in agency-hub)
```

---

## Frontend Route Map

| Path | Component | Guard |
|------|-----------|-------|
| `/login` | LoginPage | GuestRoute |
| `/auth/callback` | AuthCallbackPage | none (Supabase redirects here) |
| `/link-error` | LinkErrorPage | none (public error page) |
| `/reset-password` | ResetPasswordPage | none |
| `/welcome` | WelcomePage | AuthOnlyRoute (auth but no project required) |
| `/dashboard` | DashboardPage | ProtectedRoute (requires project) |
| `/pages` | PagesListPage | ProtectedRoute |
| `/pages/:slug` | PageEditorPage | ProtectedRoute |
| `/blog` | BlogListPage | ProtectedRoute |
| `/blog/new` | NewPostPage | ProtectedRoute |
| `/blog/:id/edit` | EditPostPage | ProtectedRoute |
| `/media` | MediaPage | ProtectedRoute |
| `/forms` | FormsPage | ProtectedRoute |
| `/settings` | SettingsPage | ProtectedRoute |
| `/assistant` | AssistantPage | ProtectedRoute |

**Route guards:**
- `GuestRoute` — redirects to `/dashboard` if already authenticated
- `AuthOnlyRoute` — requires valid portal JWT; does NOT wrap with `SupabaseProvider` (use for pages that work before a project is set up)
- `ProtectedRoute` — requires portal JWT + wraps with `SupabaseProvider` (all CMS pages)

---

## Frontend Feature Structure

```
src/
├── components/
│   ├── layout/         portal-layout.tsx, portal-sidebar.tsx, portal-header.tsx, onboarding-layout.tsx
│   ├── guards/         AuthOnlyRoute.tsx, GuestRoute.tsx, ProtectedRoute.tsx
│   ├── shared/         save-indicator.tsx
│   └── ui/             agency-badge.tsx, hard-shadow-card.tsx, status-pill.tsx
├── contexts/
│   ├── auth-context.tsx         → portal JWT, login/logout, useAuth()
│   └── supabase-context.tsx     → Supabase client per project, useTenantSupabase(), useProjectContext()
├── features/
│   ├── onboarding/     welcome-page.tsx, link-error-page.tsx
│   ├── auth/           login-page.tsx, auth-callback-page.tsx, reset-password-page.tsx
│   ├── dashboard/      dashboard-page.tsx + quick-actions, recent-edits, form-submissions-preview
│   ├── pages/          pages-list-page.tsx, page-editor-page.tsx + section editors (hero, features, about, testimonials, cta)
│   ├── blog/           blog-list-page.tsx, new-post-page.tsx, edit-post-page.tsx + tiptap-editor, post-meta-sidebar
│   ├── media/          media-page.tsx + file-browser, storage-item, media-picker-modal
│   ├── forms/          forms-page.tsx + submissions-table, submission-detail
│   ├── settings/       settings-page.tsx + general-settings, seo-settings, social-settings, nav-editor
│   └── assistant/      assistant-page.tsx + instruction-form, diff-preview, apply-bar, rate-limit-banner
├── lib/
│   ├── axios.ts        → portal backend calls (CSRF attached, 401 refresh, 429 toast)
│   └── utils.ts        → cn(), formatDate(), formatBytes(), slugify()
└── types/index.ts      → Page, Post, NavItem, FormSubmission, FieldChange, ProjectEntry, etc.
```

---

## Supabase Query Pattern (CMS reads/writes)

```typescript
export function usePages() {
  const supabase = useTenantSupabase()
  return useQuery({
    queryKey: ['pages'],
    queryFn: async () => {
      const { data, error } = await supabase.from('pages').select('*').order('slug')
      if (error) throw error
      return data
    }
  })
}
```

Never initialise a Supabase client outside of `supabase-context.tsx`. Never call `useTenantSupabase()` outside `ProtectedRoute` — it throws.

---

## Client CMS Supabase Schema

Standard tables present on every client's Supabase project (migrated on portal registration):

```sql
site_settings(id, key TEXT, value TEXT, updated_at)
pages(id, slug TEXT UNIQUE, title TEXT, sections JSONB, seo_title TEXT, seo_description TEXT, is_published BOOL, updated_at)
posts(id, slug TEXT UNIQUE, title TEXT, content TEXT, excerpt TEXT, cover_image_url TEXT,
      author_name TEXT, is_published BOOL, published_at TIMESTAMPTZ, created_at, updated_at)
nav_items(id, label TEXT, url TEXT, order INT, is_external BOOL)
form_submissions(id, form_name TEXT, data JSONB, is_read BOOL, submitted_at TIMESTAMPTZ)
media(id, filename TEXT, url TEXT, mime_type TEXT, size_bytes INT, uploaded_at TIMESTAMPTZ)
```

---

## Claude Content Assistant

**Rate limits (enforced server-side):**
- 5 req/min per `tenant_id` (Redis sliding window)
- 20 req/hour per `tenant_id` (Redis sliding window)
- Monthly token budget checked via agency-hub API

**Usage recording:** `POST /api/claude/usage` on agency-hub after every successful call. Fire-and-forget — never fail the client request if this fails.

**Response format (always a JSON array):**
```typescript
interface FieldChange {
  field:    string  // key within the section JSONB
  current:  string  // existing value
  proposed: string  // Claude's suggestion
  notes:    string  // one-sentence explanation
}
```

**Rules:**
- Never let Claude write directly. Apply is always a separate explicit client action.
- After Apply → write to client's Supabase → call `POST /api/revalidate` → portal triggers ISR on live site.
- If Claude returns invalid JSON → return 500 "temporarily unavailable".

**429 error copy:**
```
minute limit → "You're making requests too quickly. Please wait a moment."
hour limit   → "Hourly limit reached. The assistant will be available again soon."
budget       → "Your monthly content assistant limit has been reached. Your website team will be in touch."
```

---

## ISR Revalidation

After any confirmed CMS write, the portal backend triggers ISR on the client's live site (fire-and-forget):

```
POST https://[site_url]/api/revalidate
Headers: X-Revalidate-Secret: [REVALIDATE_SECRET], X-Client-ID: [tenant_id]
Body: { "paths": ["/", "/blog/slug"] }
```

Handled by `internal/revalidate/service.go`. Frontend never calls this — always triggered server-side after a confirmed mutation.

---

## Security Rules

- CSRF token required on all state-changing routes
- Rate limiting: auth routes 5/min per IP, Claude limits per tenant
- `PORTAL_ADMIN_SECRET` used for all machine-to-machine calls from agency-hub
- Service role key: backend only, never in any API response, never in JWT
- Never log Supabase credentials
- Tenant isolation: JWT `tenant_id` is the only source of truth for which `tenant_projects` rows to access
- Startup: `ValidateManagementToken` checks agency-hub is reachable before serving (`os.Exit(1)` if not)

---

## Environment Variables

```env
# Portal DB (tenant registry only)
SUPABASE_DB_URL=postgresql://postgres:password@db.<ref>.supabase.co:5432/postgres
DB_SSLMODE=require

# Agency-hub server-to-server
AGENCY_API_URL=https://agency-hub.yourdomain.com
PORTAL_ADMIN_SECRET=<shared secret — also set in agency-hub's PORTAL_ADMIN_SECRET>

# Auth
JWT_SECRET=<random 64+ chars — also used as encryption key base>
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# Redis (rate limiting)
UPSTASH_REDIS_URL=

# Claude
ANTHROPIC_API_KEY=sk-ant-...
ANTHROPIC_DEFAULT_MODEL=claude-haiku-4-5-20251001
CLAUDE_DEFAULT_MONTHLY_TOKEN_BUDGET=150000

# Email (choose one provider)
MAILER_PROVIDER=resend            # or "brevo"
EMAIL_FROM=noreply@youragency.com
RESEND_API_KEY=                   # if MAILER_PROVIDER=resend
BREVO_SMTP_USER=                  # if MAILER_PROVIDER=brevo
BREVO_SMTP_KEY=                   # if MAILER_PROVIDER=brevo

# App
ENVIRONMENT=development
PUBLIC_URL=http://localhost:8081   # used in email links
FRONTEND_URL=http://localhost:5174
PORT=8081
```

---

## Do Not

**Backend:**
- Do not write CMS content from the portal backend — the frontend Supabase client handles it
- Do not expose service role key in any API response
- Do not call the Claude API without checking rate limits + budget first
- Do not trigger ISR from the frontend — always via the portal backend after a confirmed mutation
- Do not store Supabase credentials unencrypted

**Frontend:**
- Do not initialise the Supabase client outside of `supabase-context.tsx`
- Do not call `useTenantSupabase()` outside `ProtectedRoute` — it throws
- Do not call Claude generate without showing a preview diff first
- Do not let the user apply Claude changes without an explicit confirm step
- Do not store the Claude API key in the frontend
- Do not use the old `/connect` or connection-token flow — that path no longer exists
