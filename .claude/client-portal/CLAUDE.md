> **This file mirrors the client-portal repo's `.claude/CLAUDE.md` for cross-repo context.**
> **Canonical source:** `/Users/dagi/Documents/Github/Matt x Dagim/Client-Management/.claude/CLAUDE.md`

# CLAUDE.md — client-portal
> Multi-tenant CMS dashboard. One deployment serves all agency clients.
> Each client is isolated in their own Supabase project.
> Stack: Go 1.24+ backend · React 19 + Vite frontend · Supabase

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

---

## Auth Flow (magic link, no connection_token)

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

```typescript
// src/contexts/supabase-context.tsx
// Fetches all projects for the authenticated tenant on mount.
// Exposes: supabase (client for active project), activeProject, projects, setActiveProjectId
const { data } = useQuery({ queryKey: ['projects'], queryFn: () => api.get('/api/projects') })
```

- `GET /api/projects` returns all `tenant_projects` rows (decrypted `supabase_url` + `supabase_anon_key` only)
- `setActiveProjectId` switches the active Supabase client and clears all cached CMS queries
- All CMS hooks call `useTenantSupabase()` — never initialise a Supabase client directly
- If no projects exist yet: `SupabaseProvider` shows a "no project set up" screen and polls every 10s
- `/welcome` route uses `AuthOnlyRoute` (not `ProtectedRoute`) so it works before any project is registered

---

## Portal DB Schema

```sql
tenants(
  id UUID PK,           -- = client_id from agency-hub
  onboarded_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ
)

tenant_projects(
  id UUID PK,
  tenant_id → tenants,
  agency_project_id TEXT,
  name TEXT,
  supabase_url_encrypted TEXT,
  supabase_anon_encrypted TEXT,
  supabase_service_role_encrypted TEXT,
  supabase_db_url_encrypted TEXT,
  site_url TEXT,
  created_at TIMESTAMPTZ
)

tenant_users(
  id UUID PK,
  tenant_id → tenants,
  email TEXT,
  created_at TIMESTAMPTZ,
  UNIQUE(tenant_id, email)
)

email_confirmations(
  id UUID PK,
  tenant_id → tenants,
  project_id UUID,
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
├── middleware/    JWT auth, CSRF, rate limit, CORS, security headers, logger
├── config/        LoadConfig() — all env vars
├── database/      pgxpool connection + MigrateClientDB (runs migrations on client Supabase)
└── utils/         crypto.go, errors.go, response.go, supabase.go
```

---

## Admin Endpoints (machine-to-machine, no JWT/CSRF)

All authenticated by `Authorization: Bearer <PORTAL_ADMIN_SECRET>`.

```
POST /api/admin/register-client    → RegisterClientRequest — upsert tenant + project creds + send invite
POST /api/admin/send-invite        → {client_id, email}
POST /api/admin/resend-invite      → {client_id, email}
DELETE /api/admin/deregister-client/{client_id} → remove tenant from portal
```

---

## Frontend Route Map

| Path | Component | Guard |
|------|-----------|-------|
| `/login` | LoginPage | GuestRoute |
| `/auth/callback` | AuthCallbackPage | none |
| `/link-error` | LinkErrorPage | none |
| `/reset-password` | ResetPasswordPage | none |
| `/welcome` | WelcomePage | AuthOnlyRoute |
| `/dashboard` | DashboardPage | ProtectedRoute |
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
- `AuthOnlyRoute` — requires valid portal JWT; does NOT wrap with `SupabaseProvider`
- `ProtectedRoute` — requires portal JWT + wraps with `SupabaseProvider`

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

Never initialise a Supabase client outside of `supabase-context.tsx`.
Never call `useTenantSupabase()` outside `ProtectedRoute` — it throws.

---

## Claude Content Assistant

**Rate limits (per tenant_id, Redis sliding window):**
- 5 req/min → "You're making requests too quickly. Please wait a moment."
- 20 req/hour → "Hourly limit reached. The assistant will be available again soon."
- budget → "Your monthly content assistant limit has been reached. Your website team will be in touch."

**Usage recording:** `POST /api/claude/usage` on agency-hub after every successful call. Fire-and-forget.

---

## Environment Variables

```env
SUPABASE_DB_URL=postgresql://...
DB_SSLMODE=require
AGENCY_API_URL=https://agency-hub.yourdomain.com
PORTAL_ADMIN_SECRET=<shared secret>
JWT_SECRET=<random 64+ chars>
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h
UPSTASH_REDIS_URL=
MAILER_PROVIDER=resend            # or "brevo"
EMAIL_FROM=noreply@youragency.com
RESEND_API_KEY=
ANTHROPIC_API_KEY=sk-ant-...
ANTHROPIC_DEFAULT_MODEL=claude-haiku-4-5-20251001
CLAUDE_DEFAULT_MONTHLY_TOKEN_BUDGET=150000
ENVIRONMENT=development
PUBLIC_URL=http://localhost:8081
FRONTEND_URL=http://localhost:5174
PORT=8081
```

---

## Do Not

**Backend:**
- Do not write CMS content from the portal backend
- Do not expose service role key in any API response
- Do not call the Claude API without checking rate limits + budget first
- Do not trigger ISR from the frontend
- Do not store Supabase credentials unencrypted
- Do not use the old `/connect` or connection-token flow — that path no longer exists

**Frontend:**
- Do not initialise the Supabase client outside of `supabase-context.tsx`
- Do not call `useTenantSupabase()` before auth is established (it throws)
- Do not call Claude generate without showing a preview first
- Do not let the user apply Claude changes without an explicit confirm step
- Do not store the Claude API key in the frontend
