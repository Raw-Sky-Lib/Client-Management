# CLAUDE.md — client-portal
> Multi-tenant client management dashboard. Agency-branded. One deployment, all clients.
> Clients manage their website content (pages, blog, media, SEO) from here.

---

## What This App Is

client-portal is the dashboard clients use to manage their website content. It is multi-tenant — one hosted instance serves all clients, each isolated by their own Supabase project. The portal backend is a thin layer: it validates onboarding tokens with agency-hub, proxies Claude API calls (so the key never reaches the browser), and triggers ISR revalidation on the client-site after every content update.

All CMS data (pages, posts, media, settings) lives in each client's own Supabase project. The portal reads and writes it directly from the React frontend using the Supabase JS client — the Go backend only handles operations that need server-side authority.

---

## Stack

**Backend (`api/`)**
- Go 1.24+ · Chi v5
- PostgreSQL via Supabase (portal's own project — for tenant registry + sessions only)
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

Every feature in this app touches one of two databases. Keep them separate:

```
Connection A — Agency-hub API
  Used for: token validation on onboarding, startup management token check
  How: HTTP calls to agency-hub backend (AGENCY_API_URL)
  Auth: Authorization: Bearer AGENCY_MANAGEMENT_TOKEN
  NOT used for: any CMS content read/write

Connection B — Client's Supabase project
  Used for: all CMS content (pages, posts, media, settings, form submissions)
  How: Supabase JS client initialized with CLIENT_SUPABASE_URL + CLIENT_SUPABASE_ANON_KEY
  Frontend reads: anon key with RLS
  Backend writes: service role key (stored in portal backend env only)
  NOT used for: auth or portal session management
```

When writing any feature, be explicit which connection it uses. Never mix them.

---

## Tenant Context

Because this is multi-tenant, every authenticated request carries a `tenant_id` (= `client_id` from agency-hub). This is embedded in the portal's JWT as a claim.

The portal backend injects the tenant's Supabase config (URL + anon key) into the JWT or session context so the frontend Supabase client can be initialized with the correct project credentials for this tenant.

**Frontend Supabase client initialization:**
```typescript
// src/contexts/supabase-context.tsx
// Initialized once on login using credentials from the JWT/session:
const supabase = createClient(tenantSupabaseUrl, tenantSupabaseAnonKey)
```

---

## Design → Implementation Workflow (Variants)

Same rules as agency-hub CLAUDE.md — Variants is the visual source of truth.

### Rule 1 — Preserve the design, adapt the code
When a Variants export is pasted, preserve all layout, spacing, and visual decisions. Only adapt:
- Non-shadcn UI elements → shadcn equivalents
- Untyped props → TypeScript interfaces
- Hardcoded colors → Tailwind v4 utilities
- Hardcoded content → data-driven props

### Rule 2 — First design locks the pattern
First Variants-derived card → all cards follow. First editor UI → all editors follow.

### Rule 3 — Content editors need special attention
The portal has rich content editing surfaces (page section editors, Tiptap blog editor, media picker). These are more complex than a data table. When designing them in Variants:
- Design the **empty state** (no content yet)
- Design the **populated state** (with real content)
- Design the **edit mode** (fields revealed, save/cancel visible)

### Variants Paste Prompt (portal-specific)
```
Here is a Variants export for [component name] in the client-portal.

Adapt it to portal conventions:
Stack: React 19 + TypeScript (strict) + Tailwind v4 + shadcn/ui
Target file: src/features/[feature]/components/[ComponentName].tsx

Specific portal rules:
- All Supabase reads in this component go through the tenant Supabase client
  from useTenantSupabase() context hook
- All CMS mutations should call invalidateQueries after success
- If this is a form, use React Hook Form + Zod
- If this involves saving content, show a save indicator (not just a toast)
- Preserve the layout and spacing exactly

[paste Variants code here]
```

---

## Onboarding Flow — Code Context

The most complex flow in this app. Three stages:

```
Stage 1: /connect
  Client enters: connection_token + email
  Portal backend:
    1. Calls GET /validate-management-token on agency-hub (verify portal is legit)
    2. Calls POST /validate-connection-token on agency-hub with the entered token
    3. If valid: creates user in client's Supabase Auth via service role key
    4. Sends email confirmation via Resend
  Frontend: simple form, clear error states for expired/invalid token

Stage 2: /confirm?token=...
  Client clicks email link
  Portal backend:
    1. Verifies the email confirmation token (stored in portal's own DB)
    2. Marks onboarding complete in portal tenant registry
    3. Issues portal JWT (contains: user_id, tenant_id, client_supabase_url, client_supabase_anon_key)
  Frontend: redirect to /dashboard with session established

Stage 3: /dashboard (first time)
  Client lands here with full session
  Supabase context initialized with their project credentials from JWT
  Content load begins
```

---

## Claude Content Assistant — Code Context

Full design in Master Plan Part 11. Key points for implementation:

**Rate limits enforced before every API call:**
- 5 req/min per `tenant_id` (Redis sliding window key: `claude_rl:{tenant_id}:minute`)
- 20 req/hour per `tenant_id` (Redis key: `claude_rl:{tenant_id}:hour`)
- Monthly token budget checked in agency Supabase `claude_usage` table

**Response format from Claude — always a JSON array:**
```typescript
interface FieldChange {
  field: string      // the key within the section JSONB
  current: string    // existing value
  proposed: string   // Claude's suggestion
  notes: string      // one-sentence explanation
}
```

**Preview/confirm UI:**
- Show side-by-side diff: Current | Proposed
- Client clicks "Apply" → write only changed fields to client's Supabase
- After write → call portal backend `POST /api/revalidate` → portal triggers client-site ISR
- Client clicks "Discard" → nothing happens

**Never let Claude write directly.** The apply step is always a separate client action.

---

## ISR Revalidation — After Every Content Update

After any successful content write to the client's Supabase, the portal backend must trigger the client-site to revalidate:

```
POST https://[client-site-url]/api/revalidate
Headers:
  X-Revalidate-Secret: [REVALIDATE_SECRET]
  X-Client-ID: [AGENCY_CLIENT_ID]
Body: { "paths": ["/", "/blog/[slug]"] }
```

This is handled by a `revalidate` package in the portal backend. The frontend never calls this directly — it's always triggered server-side after a confirmed content mutation.

---

## Security Rules

- CSRF token required on all state-changing routes
- Rate limiting: 5/min on onboarding, 30/min on authenticated routes, separate Claude limits
- Management token validation on startup — fail fast if invalid
- Service role key: server-side only, never returned in any API response
- Client Supabase anon key included in JWT (encrypted) — never log it
- Audit-worthy events: onboarding complete, content updates via Claude, connection token used

---

## Backend Conventions

Same Go patterns as agency-hub. Feature structure:
```
internal/<feature>/{model,repository,service,handler,routes}.go
```

Unique to portal:
- `internal/tenant/` — resolve tenant from JWT claim, provide tenant config to other services
- `internal/onboarding/` — token validation, account creation, email confirmation
- `internal/claude/` — rate limit check, prompt build, API call, usage recording
- `internal/revalidate/` — trigger client-site ISR after content mutations

---

## Frontend Conventions

Same React patterns as agency-hub, plus:

- `src/contexts/supabase-context.tsx` — provides the tenant's Supabase client
- All CMS queries use `useTenantSupabase()` hook to get the Supabase client
- Tiptap editor lives in `src/features/blog/components/PostEditor.tsx`
- Media library uploads directly to tenant's Supabase Storage via anon client

**Supabase query pattern in hooks:**
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

---

## Do Not

**Backend:**
- Do not write CMS content directly from the portal backend — the frontend Supabase client handles reads/writes, backend only handles server-side authority operations
- Do not expose the service role key in any API response
- Do not call the Claude API without checking rate limits + token budget first
- Do not trigger ISR revalidation from the frontend — always via backend after confirm

**Frontend:**
- Do not initialize the Supabase client outside of `supabase-context.tsx`
- Do not call Claude's generate endpoint without showing a preview first
- Do not let the user apply changes without an explicit confirm step
- Do not store the Claude API key anywhere in the frontend

---

## DB Schema Reference

**Portal's own Supabase (tenant registry + sessions — not CMS):**
```sql
tenants(id [= client_id from agency-hub], supabase_url_encrypted, onboarded_at, created_at)
email_confirmations(id, tenant_id, email, token_hash, expires_at, used_at, created_at)
```

**Client's Supabase (CMS — read/written by portal frontend + site):**
```sql
site_settings(id, key, value, updated_at)
pages(id, slug, title, sections JSONB, seo_title, seo_description, is_published, updated_at)
posts(id, slug, title, content, excerpt, cover_image_url, author_name, is_published,
      published_at, created_at, updated_at)
nav_items(id, label, url, order, is_external)
form_submissions(id, form_name, data JSONB, is_read, submitted_at)
media(id, filename, url, mime_type, size_bytes, uploaded_at)
```

---

## Environment Variables Reference

```env
# Portal's own Supabase (tenant registry)
SUPABASE_DB_URL=postgresql://...
DB_SSLMODE=require

# Agency-hub API connection
AGENCY_API_URL=https://agency-hub.railway.app
AGENCY_CLIENT_ID=<uuid>
AGENCY_MANAGEMENT_TOKEN=<plaintext management token>

# Client Supabase (service role — server-side only, for onboarding account creation)
# Note: per-client credentials are stored encrypted in the tenant registry DB
# and loaded from there at runtime, not from env vars

# Auth
JWT_SECRET=
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
