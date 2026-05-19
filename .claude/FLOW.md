# Client Portal — Complete Flow Reference
> One-stop reference for every user-facing and system flow.
> The old connection_token / `/connect` flow NO LONGER EXISTS. Do not reference it.

---

## 1. Tenant Registration (agency-hub → portal, per project)

Called by agency-hub's `PortalCaller.RegisterProject()` — not by the client.

```
POST /api/admin/register-client
Auth: Authorization: Bearer <PORTAL_ADMIN_SECRET>
Body: {
  client_id, project_id, project_name, email,
  client_supabase_url, client_supabase_anon_key,
  client_supabase_service_role_key, client_supabase_db_url,
  site_url
}

→ Validates Supabase credentials (test connection)
→ Encrypts all credentials (AES-256-GCM, key derived from JWT_SECRET via HKDF)
→ UPSERTs tenant row in portal DB (tenants table)
→ UPSERTs project row in portal DB (tenant_projects table)
→ Runs CMS migrations on client's Supabase (CREATE TABLE IF NOT EXISTS)
→ Creates default media storage bucket
→ Sends magic-link invite email to client via Supabase Auth
→ Creates tenant_users record (email → tenant_id mapping)
→ Returns 201 { "registered": true }
```

Triggered from agency-hub:
- Automatically in background when Supabase credentials are saved on a project
- Explicitly via the "Push to Portal" button on the project detail page

---

## 2. Client First Login (magic link)

```
1. Client receives invite email → clicks magic link
   ↓
2. Supabase Auth redirects to /auth/callback?access_token=...
   (page: AuthCallbackPage — no guard, Supabase redirects here)
   ↓
3. AuthCallbackPage extracts token → POST /api/auth/exchange
   Body: { supabase_access_token }
   ↓
4. Portal backend:
   a. Verifies Supabase access_token against Supabase Auth API
   b. Extracts email from token
   c. Looks up tenant_id via tenant_users (email → tenant_id)
   d. Issues portal JWT pair (access + refresh, HTTP-only cookies)
   e. Marks tenant.onboarded_at in portal DB (if not already set)
   ↓
5. POST /api/clients/{client_id}/portal-onboarded on agency-hub
   (Bearer PORTAL_ADMIN_SECRET — fire-and-forget callback)
   ↓
6. Redirect to /welcome
```

---

## 3. Returning Login

### Magic link
```
POST /api/auth/magic-link
Body: { email }
→ Portal looks up tenant by email in tenant_users
→ Calls Supabase Auth to send magic link to that email
→ Returns 200 (email sent)

Client clicks link → same AuthCallbackPage flow as step 2 above → /dashboard
```

### Password login
```
POST /api/auth/login
Body: { email, password }
→ Verifies email+password against client's Supabase Auth
→ Issues portal JWT pair
→ Sets cookies → return 200
```

---

## 4. Session Management

```
Portal JWT:
  access_token:  15 min, HTTP-only cookie
  refresh_token: 7 days, HTTP-only cookie

JWT claims (lean — NO Supabase credentials embedded):
  { user_id, tenant_id, email }

On app load (AuthProvider):
  1. GET /api/auth/csrf → sets csrf_token in React state
  2. GET /api/auth/profile → initializes auth context (returns user info)

On SupabaseProvider mount (inside ProtectedRoute):
  1. GET /api/projects → returns all tenant_projects rows
     (decrypted supabase_url + supabase_anon_key for each project)
  2. Initializes Supabase JS client for active project
  3. If no projects → shows "no project set up" screen, polls every 10s

Silent refresh (axios interceptor):
  On 401 → POST /api/auth/refresh → new access_token cookie → retry original

Logout: POST /api/auth/logout → clears cookies → AuthProvider sets user = null
```

---

## 5. Multi-Project Switching

```
GET /api/projects
→ Returns all tenant_projects rows for the tenant
→ Each entry: { id, agency_project_id, name, supabase_url, supabase_anon_key, site_url }
   (service_role_key NEVER returned)

setActiveProjectId(id):
→ Switches the active Supabase client
→ Clears all cached CMS queries (TanStack Query invalidation)
→ All useTenantSupabase() calls now use the new client
```

---

## 6. Route Guards

```
GuestRoute       → redirects to /dashboard if authenticated
AuthOnlyRoute    → requires portal JWT, does NOT wrap with SupabaseProvider
                   Use for: /welcome (auth required, no Supabase project yet)
ProtectedRoute   → requires portal JWT + wraps with SupabaseProvider
                   Use for: all CMS pages
```

---

## 7. Password Reset

```
POST /api/auth/reset-password/request
Body: { email }
→ Silently succeeds even if email not found
→ Sends link: {publicURL}/api/auth/reset-password/verify?token=...

GET /api/auth/reset-password/verify?token=...
→ Validates token (not yet consumed)
→ Redirects to {frontendURL}/reset-password?token=...

/reset-password?token=... (public route)
→ POST /api/auth/reset-password/confirm
Body: { token, password }
→ Validates token → updates client's Supabase Auth password → marks token used
→ Frontend shows success screen → "Back to sign in" → /login
```

---

## 8. Content Management — Pages

```
/pages → PagesListPage (ProtectedRoute)
  useTenantSupabase() → SELECT id, slug, title, is_published, updated_at FROM pages
  Click row → /pages/:slug

/pages/:slug → PageEditorPage (ProtectedRoute)
  SELECT * FROM pages WHERE slug = $1
  sections JSONB → Object.keys() → section list

  Section editor dispatch:
    hero         → HeroEditor        (headline, subheadline, cta_label, cta_url)
    features     → FeaturesEditor    (repeatable: icon, title, description)
    about        → AboutEditor       (body, image_url)
    testimonials → TestimonialsEditor (repeatable: quote, author, role, avatar)
    cta          → CTAEditor         (headline, subheadline, button_label, button_url)
    [unknown]    → GenericEditor     (key-value fallback)

  Save flow:
    1. UPDATE pages SET sections = $1, updated_at = NOW() WHERE slug = $2
    2. POST /api/revalidate { paths: ['/'] }
    3. invalidateQueries(['page', slug])
    4. SaveIndicator: saving → saved (2s) → idle

  Publish toggle:
    UPDATE pages SET is_published = $1 WHERE slug = $2
    → POST /api/revalidate
```

---

## 9. Content Management — Blog

```
/blog → BlogListPage
  SELECT id, slug, title, is_published, published_at, updated_at FROM posts

/blog/new → NewPostPage
  INSERT INTO posts → redirect to /blog/:id/edit

/blog/:id/edit → EditPostPage
  Tiptap editor — auto-save 2s debounce after typing stops
  Slug: auto-generated from title, editable, unique check on blur
  Publish toggle: sets published_at on first publish, NOT cleared on unpublish
  Cover image via MediaPickerModal
  Save: UPDATE posts → POST /api/revalidate { paths: ['/blog', '/blog/:slug'] }
```

---

## 10. Media Library

```
/media → MediaPage (ProtectedRoute)
  Storage-native: reads from Supabase Storage bucket (no media table)
  Bucket name: derived from site_url hostname (e.g. "acmecorp-com")

  FileBrowser: recursive folder tree + breadcrumb navigation
  StorageItemCard: folder or file + copy URL + delete for files

  Upload:
    → validate (jpeg/png/webp/gif/svg, max 5MB)
    → supabase.storage.from(bucket).upload(path, file)
    → get public URL

  MediaPickerModal (reusable across all editors):
    → wraps FileBrowser in selectable mode
    → onSelect(url) callback
    → used by: PostMetaSidebar (cover image), section editors
```

---

## 11. Claude Content Assistant

```
POST /api/assistant/generate (authenticated, CSRF required)
Rate limits (per tenant_id, Redis sliding window):
  5 req/min  → "You're making requests too quickly. Please wait a moment."
  20 req/hour → "Hourly limit reached. The assistant will be available again soon."
  budget     → "Your monthly content assistant limit has been reached. Your website team will be in touch."

Request: { page_slug, section_key, instruction }
Response: [{ field, current, proposed, notes }]

Apply flow:
  1. User reviews diff (DiffPreview)
  2. Clicks Apply → writes changed fields to client's Supabase
  3. → POST /api/revalidate → portal triggers ISR on live site
  4. Never write directly — always preview first, apply is a separate explicit action

Usage recording:
  Portal calls POST /api/claude/usage on agency-hub after every successful call
  Fire-and-forget — never fail the client request if this call fails
```

---

## 12. ISR Revalidation

```
After every confirmed CMS write:

Frontend never calls ISR directly.
Always: POST /api/revalidate { paths: ['/'] } to portal backend

Portal backend (internal/revalidate/service.go):
  1. Gets tenant config from context (site_url)
  2. Fire-and-forget goroutine:
     POST {site_url}/api/revalidate
     Headers: X-Revalidate-Secret, X-Client-ID: {tenant_id}
     Body: { paths }
  3. Returns { triggered: true } immediately (never blocks)

Paths to revalidate by content type:
  pages save/publish  → ['/']
  blog post save      → ['/blog', '/blog/:slug']
  nav save            → ['/']
  settings save       → ['/']
  Claude apply        → ['/'] or ['/blog/:slug'] depending on target
```

---

## 13. Settings

```
/settings → SettingsPage (tabbed)
  General tab:  site_settings key/value (site_name, tagline, logo_url, contact_email)
  SEO tab:      site_settings key/value (seo_title, seo_description, og_image_url)
  Social tab:   site_settings key "social_links" (JSON string, dynamic platforms)
  Nav tab:      nav_items table (drag-to-reorder, add/remove items)

All saves: UPDATE site_settings WHERE key = $1 → POST /api/revalidate
Nav save:  UPSERT nav_items → POST /api/revalidate
```

---

## Key Rules (Do Not Violate)

- **Two connections:** portal backend API (axios, `/api/*`) vs client Supabase (JS client). Never mix.
- **Supabase client:** only initialized inside `supabase-context.tsx`. Never call `useTenantSupabase()` outside `ProtectedRoute`.
- **Service role key:** backend only. Never in any API response. Never in JWT.
- **CSRF:** double-submit. `X-CSRF-Token` header required on all mutations.
- **Tenant isolation:** `tenant_id` from JWT is the sole source of truth for which projects to access.
- **ISR:** always via portal backend after confirmed mutation. Frontend never calls client site directly.
- **Multi-project:** all CMS hooks call `useTenantSupabase()` which resolves to the active project client.
