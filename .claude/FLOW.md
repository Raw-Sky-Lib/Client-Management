# Client Portal — Complete Flow Reference

> One-stop reference for every user-facing and system flow. Read this before building any feature.

---

## 1. Tenant Registration (agency-hub → portal, one-time per client)

Called by agency-hub when a new client is onboarded there, not by the client.

```
POST /api/admin/register-client
Auth: Bearer AGENCY_MANAGEMENT_TOKEN + X-Client-ID header
Body: { client_id, client_supabase_url, client_supabase_anon_key,
        client_supabase_service_role_key, site_url }

→ Encrypts all 3 Supabase credentials (AES-256-GCM)
→ UPSERTs into `tenants` table
→ Returns 201 { "registered": true }
```

---

## 2. Client Onboarding (first-time setup)

```
/connect  →  POST /api/onboarding/connect
  Body: { connection_token, email }
  1. Validates management token with agency-hub (startup check, not per-request)
  2. Calls POST /validate-connection-token on agency-hub
     Errors: "expired" | "used" | "invalid" | tenant not registered
  3. Stores hashed token in email_confirmations table
  4. Sends confirmation email via Resend (link: {publicURL}/api/onboarding/confirm?token=...)
  → Frontend shows CheckEmailScreen

Client clicks email link  →  GET /api/onboarding/confirm?token=...
  1. Verifies + marks token used in email_confirmations
  2. Creates user in client's Supabase Auth (service role key)
  3. Marks tenant.onboarded_at in portal DB
  4. Creates tenant_users record (email → tenant_id mapping)
  5. Issues portal JWT pair (access + refresh cookies)
  → Redirects to /welcome

/welcome (ProtectedRoute)
  - Optional: set password via POST /api/auth/set-password (authenticated)
  - "Go to dashboard" → /dashboard
```

**Error states:** `/link-error?reason=expired|used|invalid|error`
- `used` → "Account already verified. Sign in →"
- `expired` → "Use your access code again →"

---

## 3. Login

### Password login (primary)
```
POST /api/auth/login
Body: { email, password }
1. Looks up tenant by email (tenant_users JOIN tenants)
2. POST /auth/v1/token?grant_type=password on client's Supabase
3. Issues portal JWT pair (access_token + refresh_token cookies)
→ Frontend calls refresh() → navigate('/dashboard')
```

### Magic link (backend exists, not exposed in current UI)
```
POST /api/auth/magic-link → generates portal-native token → emails link
GET  /api/auth/login/verify?token=... → validates token → sets cookies → redirect /dashboard
```

---

## 4. Password Reset

```
Login page → "Forgot password?" → ResetRequestForm

POST /api/auth/reset-password/request
Body: { email }
→ Silently succeeds even if email not registered (no enumeration)
→ Sends email with link: {publicURL}/api/auth/reset-password/verify?token=...

GET /api/auth/reset-password/verify?token=...
→ Validates token (not consumed yet)
→ Redirects to {frontendURL}/reset-password?token=...

/reset-password?token=... (public route)
POST /api/auth/reset-password/confirm
Body: { token, password }
→ Validates token → updates Supabase Auth password → marks token used
→ Frontend shows success screen → "Back to sign in" → /login
```

---

## 5. Session Management

```
JWT access token:  15 min, HTTP-only cookie (access_token)
JWT refresh token: 7 days, HTTP-only cookie (refresh_token)

JWT claims: { user_id, tenant_id, email, supabase_url, supabase_anon_key, site_url }

On app load (AuthProvider):
  1. GET /api/auth/csrf        → sets csrf_token cookie (JS-readable)
  2. GET /api/auth/profile     → returns PortalUser, initializes auth context
  → SupabaseProvider initialized with supabase_url + supabase_anon_key from profile

Silent refresh (axios interceptor):
  On 401 → POST /api/auth/refresh → new access_token cookie → retry original request

Logout: POST /api/auth/logout → clears both cookies → setUser(null)
```

---

## 6. Content Management — Pages (built)

```
/pages → PagesListPage
  useTenantSupabase() → SELECT id, slug, title, is_published, updated_at FROM pages
  Click row → /pages/:slug

/pages/:slug → PageEditorPage
  SELECT * FROM pages WHERE slug = $1
  sections JSONB → Object.keys() = section list (dynamic per client)

  Section editor dispatch:
    hero         → HeroEditor        (headline, subheadline, cta_label, cta_url)
    features     → FeaturesEditor    (repeatable: icon, title, description)
    about        → AboutEditor       (body, image_url)
    testimonials → TestimonialsEditor (repeatable: quote, author, role, avatar)
    cta          → CTAEditor         (headline, subheadline, button_label, button_url)
    [unknown]    → GenericEditor     (key-value fallback)

  Save flow:
    1. UPDATE pages SET sections = $1, updated_at = NOW() WHERE slug = $2
    2. POST /api/revalidate { paths: ['/'] }  ← portal backend → client site ISR
    3. invalidateQueries(['page', slug]) + invalidateQueries(['pages'])
    4. SaveIndicator: saving → saved (2s) → idle

  Publish toggle:
    UPDATE pages SET is_published = $1 WHERE slug = $2
    → POST /api/revalidate
```

---

## 7. Content Management — NOT YET BUILT

| CLI  | Feature                                  |
|------|------------------------------------------|
| 28   | Blog list + post editor design           |
| 29   | BlogListPage, PostsTable                 |
| 30   | PostEditor (Tiptap), auto-save, slug gen |
| 31   | Media library design                     |
| 32   | MediaPage, MediaGrid, MediaUploader      |
| 33   | MediaPickerModal (reusable)              |
| 34   | Claude assistant design                  |
| 35   | use-assistant hook, 429 handling         |
| 36   | AssistantPanel, DiffPreview, ApplyBar    |
| 37   | FormsPage, SubmissionsTable              |
| 38   | SettingsPage (General, SEO, Nav)         |
| 39   | DashboardPage, QuickActions              |

---

## 8. ISR Revalidation (POST /api/revalidate)

```
Frontend calls: POST /api/revalidate { paths: ['/'] }
Backend:
  1. Gets tenant config from context (site_url, service_role_key)
  2. Fire-and-forget goroutine:
     POST {site_url}/api/revalidate
     Headers: X-Revalidate-Secret, X-Client-ID
     Body: { paths }
  3. Returns { triggered: true } immediately (never blocks)

Always fire after: section save, publish toggle, blog post save/publish,
                   nav save, settings save — any CMS mutation.
```

---

## 9. Claude Content Assistant (backend built, UI not yet)

```
POST /api/assistant/generate (authenticated)
Rate limits (per tenant_id):
  5 req/min  → "You're making requests too quickly. Please wait a moment."
  20 req/hour → "Hourly limit reached. The assistant will be available again soon."
  budget     → "Your monthly content assistant limit has been reached..."

Response: [{ field, current, proposed, notes }]
Apply flow (when UI built):
  User reviews diff → clicks Apply → writes fields to client Supabase → POST /api/revalidate
  Never write directly — always show preview first, apply is a separate action.
```

---

## Key Rules

- **Two connections:** agency-hub API (management token) vs client Supabase (anon key). Never mix.
- **Service role key:** server-side only. Never in any API response. Never in JWT.
- **CSRF:** double-submit pattern. `csrf_token` cookie (JS-readable) → `X-CSRF-Token` header. All mutations require it.
- **Tenant isolation:** `tenant_id` from JWT is the sole source of truth for which Supabase project to access.
- **ISR:** always via portal backend after confirmed mutation. Frontend never calls client site directly.
