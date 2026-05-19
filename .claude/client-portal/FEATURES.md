> **Canonical source:** See `Client-Management/.claude/CLAUDE.md` and `Client-Management/.claude/FLOW.md`

# client-portal — Feature Summary (Current State)

## What Exists

**Auth & Onboarding:**
- Magic link invite flow (no connection_token, no `/connect` page)
- `POST /api/admin/register-client` — agency-hub registers tenant + runs migrations + sends invite
- `POST /api/auth/exchange` — exchanges Supabase access_token for portal JWT
- `POST /api/auth/magic-link` — sends magic link for returning logins
- `POST /api/auth/login` — password login
- `GET /api/auth/profile`, `POST /api/auth/refresh`, `POST /api/auth/logout`
- `POST /api/auth/reset-password/request/verify/confirm`

**Portal JWT (lean — no Supabase creds embedded):**
```go
type PortalClaims struct {
    UserID   string `json:"user_id"`
    TenantID string `json:"tenant_id"`
    Email    string `json:"email"`
    jwt.RegisteredClaims
}
```

**Projects:**
- `GET /api/projects` — returns all tenant_projects rows (decrypted url + anon_key only)
- Used by `SupabaseProvider` on mount to initialize the Supabase JS client

**CMS (all via Supabase JS client in browser):**
- Pages + section editors (hero, features, about, testimonials, cta, generic)
- Blog (Tiptap editor, auto-save, publish toggle, cover image)
- Media library (Storage-native, folder tree, MediaPickerModal)
- Form submissions inbox (mark as read)
- Settings (general, SEO, social, nav editor)
- Dashboard (quick actions, recent edits, form submissions preview)

**Claude Assistant:**
- `POST /api/assistant/generate` — rate-limited (5/min, 20/hr, budget)
- Preview diff → explicit Apply → write to Supabase → ISR revalidation

**ISR Revalidation:**
- `POST /api/revalidate` — fires non-blocking goroutine to client site

**Admin endpoints (M2M, Bearer PORTAL_ADMIN_SECRET):**
- `POST /api/admin/register-client`
- `POST /api/admin/send-invite`
- `POST /api/admin/resend-invite`
- `DELETE /api/admin/deregister-client/{client_id}`

## What No Longer Exists

- `/connect` page — replaced by magic link invite flow
- Connection token validation — portal no longer uses management tokens
- JWT with Supabase credentials embedded — now fetched via `GET /api/projects`

See `Client-Management/.claude/FLOW.md` for detailed flow documentation.
