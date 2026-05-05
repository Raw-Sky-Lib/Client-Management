# CLAUDE.md — client-portal
> Multi-tenant client management dashboard. Agency-branded. One deployment, all clients.
> Clients manage their website content (pages, blog, media, SEO, form submissions) from here.
> Stack: Go 1.24+ backend · React 19 + Vite frontend · Supabase (portal's own + per-tenant)

---

## What This App Is

client-portal is the CMS dashboard that agency clients use to manage their website content. It is multi-tenant — one hosted instance serves all clients, each isolated by their own Supabase project.

The portal backend is a thin authority layer:
- Validates onboarding tokens with agency-hub
- Proxies Claude API calls (API key never reaches the browser)
- Triggers ISR revalidation on the client-site after every content update

All CMS data (pages, posts, media, settings) lives in each client's own Supabase project. The portal reads and writes it directly from the React frontend using the Supabase JS client — the Go backend only handles operations that require server-side authority.

---

## Stack

**Backend (`api/`)**
- Go 1.24+ · Chi v5
- PostgreSQL via Supabase (portal's own project — tenant registry + sessions only)
- JWT: 15-min access, 7-day refresh, HTTP-only cookies
- Upstash Redis: rate limiting + Claude per-client rate limits (minute + hour windows)
- Resend: email confirmation on onboarding
- Anthropic SDK (Go): Claude Haiku — content assistant proxy
- Swagger (swag) · slog · go-playground/validator v10

**Frontend (`web/`)**
- React 19 · TypeScript (strict) · Vite
- Tailwind CSS v4 · shadcn/ui · Radix UI
- TanStack Query v5 · React Router v7
- React Hook Form + Zod · Framer Motion · Sonner · Lucide React
- Supabase JS client (client-side, anon key — per-tenant, initialized per session)
- Tiptap: rich text editor for blog posts

---

## Two Connections — Always in Mind

Every feature touches one of two databases. Never mix them.

```
Connection A — Agency-hub API
  Used for: token validation on onboarding, startup management token check, Claude usage recording
  How: HTTP calls to agency-hub backend (AGENCY_API_URL)
  Auth: Authorization: Bearer AGENCY_MANAGEMENT_TOKEN + X-Client-ID header
  NOT used for: any CMS content read/write

Connection B — Client's Supabase project
  Used for: all CMS content (pages, posts, media, settings, form submissions)
  How: Supabase JS client initialized with CLIENT_SUPABASE_URL + CLIENT_SUPABASE_ANON_KEY
  Frontend reads/writes: anon key with RLS
  Backend writes: service role key (stored encrypted in portal DB, never returned in responses)
  NOT used for: portal auth or session management
```

When writing any feature, be explicit about which connection it uses.

---

## Tenant Context

Every authenticated request carries a `tenant_id` (= `client_id` from agency-hub), embedded in the portal JWT as a claim.

The portal JWT contains the tenant's Supabase config (URL + anon key) so the frontend Supabase client can be initialized with the correct project credentials for this tenant.

**Frontend Supabase client initialization:**
```typescript
// src/contexts/supabase-context.tsx
// Initialized once on login using credentials from the JWT:
const supabase = createClient(tenantSupabaseUrl, tenantSupabaseAnonKey)
```

---

## DB Schema

### Portal's Own Supabase (tenant registry + sessions — NOT CMS)

```sql
-- 001_create_tenants.sql
CREATE TABLE tenants (
    id                              UUID PRIMARY KEY,  -- same as client_id from agency-hub
    supabase_url_encrypted          TEXT NOT NULL,
    supabase_anon_encrypted         TEXT NOT NULL,
    supabase_service_role_encrypted TEXT NOT NULL,
    site_url                        TEXT,              -- client's live site URL (for ISR)
    onboarded_at                    TIMESTAMPTZ,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 002_create_email_confirmations.sql
CREATE TABLE email_confirmations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON email_confirmations(token_hash);
CREATE INDEX ON email_confirmations(tenant_id);

-- 003_create_tenant_users.sql
CREATE TABLE tenant_users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);
```

### Client's Supabase (CMS — read/written by portal frontend + client-site)

```sql
-- Standard tables present on every client site:
site_settings(id, key, value, updated_at)
pages(id, slug, title, sections JSONB, seo_title, seo_description, is_published, updated_at)
posts(id, slug, title, content, excerpt, cover_image_url, author_name,
      is_published, published_at, created_at, updated_at)
nav_items(id, label, url, order, is_external)
form_submissions(id, form_name, data JSONB, is_read, submitted_at)
media(id, filename, url, mime_type, size_bytes, uploaded_at)
```

---

## Credential Encryption

Client Supabase credentials are stored encrypted in the portal DB.

- Algorithm: AES-256-GCM
- Key: derived from `JWT_SECRET` via HKDF-SHA256 to 32 bytes
- `EncryptString(plaintext, key) (ciphertext string, err error)` — returns base64-encoded
- `DecryptString(ciphertext, key) (plaintext string, err error)`
- File: `internal/utils/crypto.go`

---

## Onboarding Flow

Three stages. The most complex flow in the app.

```
Stage 1: /connect
  Client enters: connection_token + email
  Portal backend:
    1. Calls GET /validate-management-token on agency-hub (verify portal is legit)
    2. Calls POST /validate-connection-token on agency-hub with the entered token
    3. If valid: stores email confirmation in portal DB (does NOT create Supabase user yet)
    4. Sends confirmation email via Resend
  Rate limit: 5 req/min per IP (brute-force protection)

Stage 2: /confirm?token=...
  Client clicks email link
  Portal backend:
    1. Verifies the email confirmation token (hashed, stored in portal DB)
    2. Creates user in client's Supabase Auth via service role key (fetched from tenant record)
    3. Marks onboarding complete in portal tenant registry
    4. Creates tenant_users record (email → tenant_id mapping for future logins)
    5. Issues portal JWT (contains: user_id, tenant_id, supabase_url, supabase_anon_key)
  Frontend: redirect to /dashboard

Stage 3: /dashboard (first time)
  Client lands with full session
  Supabase context initialized with their project credentials from JWT
  Content loading begins
```

**Register Tenant endpoint (called by agency-hub, not clients):**
```go
// POST /api/admin/register-client
// Auth: Authorization: Bearer <AGENCY_MANAGEMENT_TOKEN> + X-Client-ID
type RegisterClientRequest struct {
    ClientID                    string `json:"client_id"                     validate:"required,uuid"`
    ClientSupabaseURL           string `json:"client_supabase_url"           validate:"required,url"`
    ClientSupabaseAnonKey       string `json:"client_supabase_anon_key"      validate:"required"`
    ClientSupabaseServiceRoleKey string `json:"client_supabase_service_role_key" validate:"required"`
    SiteURL                     string `json:"site_url"                      validate:"required,url"`
}
// Encrypts credentials → UPSERTs into tenants table → returns 201 { "registered": true }
```

**Connect endpoint error messages:**
```go
// From agency-hub validate-connection-token response:
"expired"  → "Your access code has expired. Ask your website team for a new one."
"used"     → "This access code has already been used. Contact your website team."
"invalid"  → "Invalid access code. Check for typos and try again."
// Not registered in portal:
           → "This client is not set up in the portal yet. Contact your website team."
```

---

## Portal Auth — JWT Claims

```go
type PortalClaims struct {
    UserID                string `json:"user_id"`
    TenantID              string `json:"tenant_id"`          // client_id from agency-hub
    Email                 string `json:"email"`
    ClientSupabaseURL     string `json:"supabase_url"`
    ClientSupabaseAnonKey string `json:"supabase_anon_key"`  // anon key — safe to embed
    jwt.RegisteredClaims
}
```

**Magic Link Login flow:**
1. `POST /api/auth/magic-link` — portal backend calls Supabase Auth magiclink endpoint
2. User clicks link → Supabase redirects to `/auth/callback?access_token=...`
3. Frontend sends token to `POST /api/auth/exchange`
4. Portal backend verifies Supabase token, looks up tenant by email from `tenant_users`, issues portal JWT
5. Cookies set, redirect to `/dashboard`

Never store service role key in JWT. Never log anon key.

---

## Claude Content Assistant

**Rate limits (enforced server-side before every call):**
- 5 req/min per `tenant_id` — Redis key: `claude_rl:{tenant_id}:minute`
- 20 req/hour per `tenant_id` — Redis key: `claude_rl:{tenant_id}:hour`
- Monthly token budget checked via agency-hub API (`GET /api/claude/budget/{client_id}`)

**Usage recording:** Call `POST /api/claude/usage` on agency-hub (management token auth) after every successful Claude call. Fire-and-forget — never fail the request if this fails.

**Claude response format — always a JSON array:**
```typescript
interface FieldChange {
  field:    string  // key within the section JSONB
  current:  string  // existing value
  proposed: string  // Claude's suggestion
  notes:    string  // one-sentence explanation
}
```

**Rules:**
- Never let Claude write directly. Apply is always a separate client action.
- After Apply → write changed fields to client's Supabase → call `POST /api/revalidate` on portal backend → portal triggers client-site ISR.
- If Claude returns invalid JSON → return 500 "temporarily unavailable" (not a raw parse error).

**429 error messages:**
```
minute limit  → "You're making requests too quickly. Please wait a moment."
hour limit    → "Hourly limit reached. The assistant will be available again soon."
budget        → "Your monthly content assistant limit has been reached. Your website team will be in touch."
```

---

## ISR Revalidation — After Every Content Update

After any confirmed content write to the client's Supabase, the portal backend triggers ISR on the client-site:

```
POST https://[client-site-url]/api/revalidate
Headers:
  X-Revalidate-Secret: [REVALIDATE_SECRET from tenant config]
  X-Client-ID: [tenant_id]
Body: { "paths": ["/", "/blog/slug"] }
```

- Handled by `internal/revalidate/service.go`
- Non-blocking: fire-and-forget with error logging
- Frontend never calls this directly — always triggered server-side
- `site_url` stored in portal's tenant record (set during register-client)

---

## Backend Feature Structure

```
internal/<feature>/{model,repository,service,handler,routes}.go
```

Feature packages:
- `internal/startup/` — `ValidateManagementToken()` — called in main.go before serving. `os.Exit(1)` if invalid.
- `internal/onboarding/` — token validation, Supabase Auth user creation, email confirmation
- `internal/auth/` — magic link, token exchange, JWT issue/refresh/logout
- `internal/tenant/` — resolve tenant from JWT claim, decrypt and provide Supabase config
- `internal/claude/` — rate limit, budget check, prompt build, API call, usage recording
- `internal/revalidate/` — trigger client-site ISR after content mutations
- `internal/config/`, `internal/database/`, `internal/middleware/`, `internal/utils/`

**Startup validation (runs before HTTP server):**
```go
// Retry 3 times with 2s backoff (Railway cold start protection)
if err := startup.ValidateManagementToken(cfg, httpClient); err != nil {
    slog.Error("startup validation failed", "error", err)
    os.Exit(1)
}
```

---

## Frontend Feature Structure

```
src/
├── components/
│   ├── layout/         PortalLayout, PortalSidebar, PortalHeader
│   ├── guards/         ProtectedRoute, GuestRoute
│   └── shared/         SaveIndicator, EmptyState, ConfirmDialog
├── contexts/
│   ├── auth-context.tsx
│   └── supabase-context.tsx   ← tenant Supabase client lives here
├── features/
│   ├── onboarding/     ConnectPage, ConnectForm, CheckEmailScreen
│   ├── dashboard/      DashboardPage, QuickActions, RecentEdits
│   ├── pages/          PagesListPage, PageEditorPage, SectionEditor + editors
│   ├── blog/           BlogListPage, NewPostPage, EditPostPage, PostEditor (Tiptap)
│   ├── media/          MediaPage, MediaGrid, MediaUploader, MediaPickerModal
│   ├── forms/          FormsPage, SubmissionsTable, SubmissionDetail
│   ├── settings/       SettingsPage, GeneralSettings, SeoSettings, NavEditor
│   └── assistant/      AssistantPanel, InstructionForm, DiffPreview, ApplyBar
├── lib/
│   ├── axios.ts        Portal backend calls
│   └── utils.ts        cn(), formatDate(), formatBytes()
└── types/index.ts      Page, Post, NavItem, FormSubmission, FieldChange, etc.
```

**Supabase query pattern:**
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

**Shared components:**
- `SaveIndicator` — states: idle | saving | saved | error. Auto-clears "saved" after 2s.
- `EmptyState({ icon, title, description, action })`
- `ConfirmDialog({ title, description, onConfirm, dangerous? })` — wraps shadcn AlertDialog

---

## Route Map

| Path | Component | Guard |
|------|-----------|-------|
| `/connect` | ConnectPage | GuestRoute |
| `/auth/callback` | AuthCallbackPage | GuestRoute |
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

---

## Page & Section Editors

Pages JSONB sections are edited per-section. Section type → editor component:

```typescript
const sectionEditors = {
  hero:         HeroEditor,        // headline, subheadline, cta_label, cta_url
  features:     FeaturesEditor,    // repeatable: icon, title, description
  about:        AboutEditor,       // body textarea + optional image
  testimonials: TestimonialsEditor, // repeatable: quote, author, role, avatar
  cta:          CTAEditor,         // headline, subheadline, button label, button url
}
```

Each section editor:
- Shows current values in editable fields
- "Save section" button → `UPDATE pages SET sections = jsonb_set(...) WHERE slug = $1`
- After save → call `POST /api/revalidate` via portal backend with path `/`
- Shows `SaveIndicator`

---

## Blog Editor (Tiptap)

```typescript
const editor = useEditor({
  extensions: [
    StarterKit.configure({ heading: { levels: [2, 3] } }),
    Link.configure({ openOnClick: false }),
    Image,
    Placeholder.configure({ placeholder: 'Start writing...' }),
  ],
  content: post?.content ?? '',
  onUpdate: ({ editor }) => setContent(editor.getHTML()),
})
```

- Auto-save: debounce 2s after typing stops
- Slug: auto-generated from title via `slugify()`, editable, unique check on blur
- Draft/Publish toggle: `published_at` set on first publish, NOT cleared on unpublish
- Cover image via `MediaPickerModal`

---

## Media Library

- Upload to Supabase Storage `media` bucket directly from browser (anon key + Storage RLS)
- Accepts: jpeg, png, webp, gif, svg · Max: 5MB
- On upload: validate → `supabase.storage.from('media').upload()` → get public URL → insert `media` table row
- `MediaPickerModal` — reusable modal, `onSelect(url)` callback. Used by all content editors.

---

## Security Rules

- CSRF token required on all state-changing routes
- Rate limiting: 5/min on onboarding endpoints, 30/min on authenticated routes, separate Claude limits
- Management token validated on startup — `os.Exit(1)` if invalid
- Service role key: server-side only, never returned in any API response, never in JWT
- Client Supabase anon key included in JWT (HTTP-only cookie — browser cannot read raw cookie)
- Never log tenant credentials
- Tenant isolation: JWT tenant_id claim is the only source of truth for which Supabase project to access

---

## Environment Variables

```env
# Portal's own Supabase (tenant registry)
SUPABASE_DB_URL=postgresql://...
DB_SSLMODE=require

# Agency-hub API
AGENCY_API_URL=https://agency-hub.yourdomain.com
AGENCY_CLIENT_ID=<uuid>
AGENCY_MANAGEMENT_TOKEN=<plaintext management token>

# Auth
JWT_SECRET=<random 64+ chars>
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# Redis (Claude rate limiting)
UPSTASH_REDIS_URL=
UPSTASH_REDIS_TOKEN=

# Claude
ANTHROPIC_API_KEY=sk-ant-...
ANTHROPIC_DEFAULT_MODEL=claude-haiku-4-5-20251001
CLAUDE_DEFAULT_MONTHLY_TOKEN_BUDGET=150000

# Email
RESEND_API_KEY=
RESEND_FROM=noreply@youragency.com

# App
ENVIRONMENT=development
FRONTEND_URL=http://localhost:5174
PORT=8081
```

---

## Design → Implementation Workflow (Variants)

Variants is the visual source of truth for this project.

**Rule 1:** Preserve all layout, spacing, and visual decisions from the Variants export. Only adapt: non-shadcn elements → shadcn equivalents, untyped props → TypeScript interfaces, hardcoded colors → Tailwind v4 utilities, hardcoded content → data-driven props.

**Rule 2:** First Variants-derived card → all cards follow. First editor UI → all editors follow.

**Rule 3:** Content editors need three states designed: empty, populated, edit mode.

**Variants paste prompt:**
```
Here is a Variants export for [component name] in the client-portal.

Adapt it to portal conventions:
Stack: React 19 + TypeScript (strict) + Tailwind v4 + shadcn/ui
Target file: src/features/[feature]/components/[ComponentName].tsx

Portal rules:
- All Supabase reads use useTenantSupabase() context hook
- All CMS mutations call invalidateQueries after success
- If this is a form, use React Hook Form + Zod
- If this involves saving content, show SaveIndicator (not just a toast)
- Preserve the layout and spacing exactly

[paste Variants code here]
```

---

## Build Order (Milestones)

```
M1: Backend Foundation
  - Go project setup + dependencies
  - Portal's own Supabase (run migrations)
  - Startup management token validation
  - Config, DB, middleware stack (auth, CSRF, rate limit, logger)
  - Onboarding flow (connect + confirm)
  - Portal auth (magic link, exchange, JWT with tenant claims)
  - Tenant registry service (encrypt/decrypt credentials)
  - ISR revalidation service

M2: Claude Content Assistant (backend)
  - Rate limiter (Redis sliding window)
  - Usage repository (calls agency-hub API)
  - Prompt builder (fetches section from client Supabase)
  - Claude service + handler (POST /assistant/generate)

M3: Frontend Shell + Onboarding UI
  - React + Vite + TypeScript setup
  - axios client + Supabase context (useTenantSupabase)
  - Auth context + routes (ProtectedRoute, GuestRoute)
  - Design: Connect page (Variants)
  - Build: ConnectPage, ConnectForm, CheckEmailScreen
  - Design: Portal shell — sidebar + header (Variants)
  - Build: PortalLayout, PortalSidebar, PortalHeader

M4: Page & Section Editors
  - Design: Pages list + page editor layout (Variants)
  - Build: PagesListPage + PagesList
  - Build: PageEditorPage, SectionEditor, HeroEditor + SaveIndicator
  - Build: FeaturesEditor, AboutEditor, TestimonialsEditor, CTAEditor

M5: Blog Editor
  - Design: Blog list + post editor (Variants)
  - Build: BlogListPage, PostsTable
  - Build: PostEditor (Tiptap), PostMetaForm, PostStatusToggle

M6: Media Library
  - Design: Media grid (Variants)
  - Build: MediaPage, MediaGrid, MediaUploader, MediaItem
  - Build: MediaPickerModal (reusable across all editors)

M7: Claude Assistant UI
  - Design: Assistant panel + diff preview (Variants)
  - Build: use-assistant.ts hook + 429 error mapping
  - Build: InstructionForm, DiffPreview, ApplyBar, RateLimitBanner, AssistantPanel

M8: Secondary Features
  - Build: FormsPage, SubmissionsTable, SubmissionDetail
  - Build: SettingsPage (General, SEO, Social tabs), NavEditor
  - Build: DashboardPage, QuickActions, RecentEdits, FormSubmissionsPreview

M9: QA & Deploy
  - End-to-end: full onboarding flow
  - End-to-end: content edit → ISR → live on client site
  - End-to-end: Claude assistant → apply → live
  - Rate limit verification (all three limits)
  - Security review (management token, service role key, CSRF, tenant isolation)
  - Deploy: backend to Railway, frontend to Vercel
```

---

## Do Not

**Backend:**
- Do not write CMS content directly from the portal backend — the frontend Supabase client handles it
- Do not expose the service role key in any API response
- Do not call the Claude API without checking rate limits + token budget first
- Do not trigger ISR revalidation from the frontend — always via backend after confirmed mutation

**Frontend:**
- Do not initialize the Supabase client outside of `supabase-context.tsx`
- Do not call Claude's generate endpoint without showing a preview first
- Do not let the user apply changes without an explicit confirm step
- Do not store the Claude API key anywhere in the frontend
- Do not use `useTenantSupabase()` before auth is established (it throws — do not silence it)

---

## Linear Project Reference

```
Team:       client-portal
Identifier: CP  (issues: CP-1, CP-2, ...)
Labels:     backend · frontend · design · infra · security · bug · chore

Cycles:
  Cycle 1: M1 — Backend foundation
  Cycle 2: M2 — Claude backend
  Cycle 3: M3 — Frontend shell + onboarding
  Cycle 4: M4 + M5 — Content editors
  Cycle 5: M6 + M7 — Media + assistant
  Cycle 6: M8 + M9 — Secondary + QA
```

See `LINEAR-SETUP.md` in this folder for the full issue list (CP-1 through CP-41).
