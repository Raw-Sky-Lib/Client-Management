# TODO Checklist — client-portal (CLI-5 → CLI-45)

---

## Cycle 1 · M1: Backend Foundation

- [x] **CLI-5** Set up Go project + install dependencies
  - [x] Chi v5, validator, swag, air, anthropic-sdk-go, logging with slog(refer,the logging.md file)
  - [x] Create all `internal/` feature folders

- [x] **CLI-6** Set up portal's own Supabase project
  - [x] Create Supabase project: client-portal-prod
  - [x] Run migrations: `001_create_tenants.sql`, `002_create_email_confirmations.sql`, `003_create_tenant_users.sql`
  - [x] Client CMS migrations: `001–006` (site_settings, pages, posts, nav_items, form_submissions, media)
  - [x] `MigratePortalDB()` + `MigrateClientDB()` — embedded via `embed.FS`, runs on `register-client`
  - [x] `ValidateSupabaseCredentials()` — tests URL + service role key before storing

- [x] **CLI-7** Implement startup management token validation
  - [x] `internal/startup/validate.go`
  - [x] Calls `GET /validate-management-token` on agency-hub at startup
  - [x] Retry 3× with 2s backoff (Railway cold start protection)
  - [x] `os.Exit(1)` if invalid — fail loud, not silent

- [x] **CLI-8** Implement config, DB, middleware stack
  - [x] `config.go`, `db.go`, `security.go`, `logger.go`, `csrf.go`, `ratelimit.go`, `auth.go`

- [x] **CLI-9**   
  - [x] `internal/onboarding/` — full feature
  - [x] `POST /onboarding/connect`:
    1. Call agency-hub `POST /validate-connection-token` with entered token
    2. If valid: store email confirmation in portal DB (do NOT create Supabase user yet)
    3. Send confirmation email via Resend
  - [x] `GET /onboarding/confirm?token=...`:
    1. Verify + consume confirmation token from portal DB
    2. Create user in client's Supabase Auth via service role key
    3. Mark tenant onboarded in portal tenant registry
    4. Create `tenant_users` record (email → tenant_id)
    5. Issue portal JWT with tenant Supabase config embedded
  - [x] Error messages: expired / used / invalid token → correct user-facing copy
  - [x] Rate limit: 5 req/min per IP on `/connect`
  - [ ] Depends on: CLI-7, CLI-8

- [x] **CLI-10** Implement portal auth (JWT with tenant claims)
  - [x] `internal/auth/` — full feature
  - [x] `PortalClaims` struct: `user_id`, `tenant_id`, `email`, `supabase_url`, `supabase_anon_key`
  - [x] Magic link flow: `POST /auth/magic-link` → Supabase → `/auth/callback` → `POST /auth/exchange` → portal JWT
  - [x] `GET /auth/profile`, `POST /auth/refresh`, `POST /auth/logout`, `GET /auth/csrf`
  - [x] Depends on: CLI-9

- [x] **CLI-11** Implement tenant registry service
  - [x] `internal/tenant/` — model, repository, service
  - [x] AES-256-GCM encrypt/decrypt for Supabase credentials (`internal/utils/crypto.go`)
  - [x] `ResolveTenant()` middleware — inject tenant config into handler context
  - [x] `POST /api/admin/register-client` endpoint (called by agency-hub, not clients)
  - [x] Depends on: CLI-10

- [x] **CLI-12** Implement ISR revalidation service
  - [x] `internal/revalidate/service.go`
  - [x] `TriggerISR(siteURL, paths, secret)` — HTTP POST to client-site `/api/revalidate`
  - [x] Non-blocking: fire-and-forget with error logging
  - [x] Depends on: CLI-8

---

## Cycle 2 · M2: Claude Content Assistant (Backend)

- [x] **CLI-13** Implement Claude rate limiter
  - [x] `internal/claude/ratelimit.go`
  - [x] Sliding window: 5/min + 20/hour per `tenant_id`
  - [x] Redis keys: `claude_rl:{tenant_id}:minute`, `claude_rl:{tenant_id}:hour`
  - [x] Depends on: CLI-8

- [x] **CLI-14** Implement Claude usage repository
  - [x] `internal/claude/repository.go`
  - [x] `RecordUsage()` — calls `POST /api/claude/usage` on agency-hub (fire-and-forget)
  - [x] `CheckBudget()` — calls `GET /api/claude/budget/{client_id}` on agency-hub
  - [x] Auth: `Authorization: Bearer AGENCY_MANAGEMENT_TOKEN` + `X-Client-ID` header
  - [x] Never fail the user request if usage recording fails
  - [x] Depends on: CLI-11

- [x] **CLI-15** Implement Claude prompt builder
  - [x] `internal/claude/prompt.go`
  - [x] Fetches current section from client's Supabase via service role key
  - [x] Builds system prompt with: business name, page, section, current content JSON
  - [x] Returns `(systemPrompt string, currentContent map[string]any, err)`
  - [x] Depends on: CLI-11

- [x] **CLI-16** Implement Claude service + handler
  - [x] `internal/claude/service.go` — orchestrates: rate limit → budget → prompt → API call → record usage
  - [x] `internal/claude/handler.go` — `POST /assistant/generate`
  - [x] Parse Claude response as `[]FieldChange` JSON array
  - [x] Invalid JSON from Claude → return 500 "temporarily unavailable" (not raw parse error)
  - [x] 429 responses: distinct messages for minute / hour / budget exceeded
  - [x] Depends on: CLI-13, CLI-14, CLI-15

---

## Cycle 3 · M3: Frontend Shell + Onboarding UI

- [x] **CLI-17** Set up React + Vite + TypeScript project
  - [x] Same dependencies as agency-hub `web/`
  - [x] Add: Supabase JS client, Tiptap + extensions

- [x] **CLI-18** Set up axios client + Supabase context
  - [x] `src/lib/axios.ts` — portal backend calls
  - [x] `src/contexts/supabase-context.tsx` — initialize tenant Supabase client from JWT claims
  - [x] `useTenantSupabase()` hook

- [x] **CLI-19** Set up auth context + routes
  - [x] `auth-context.tsx` with `tenant_id` claim
  - [x] `routes.tsx` with `ProtectedRoute` + `GuestRoute` guards

- [x] **CLI-20** Design: Connect (onboarding) page (Variants),
  - [x] Clean centered layout
  - [x] Token field + email field
  - [x] Three states: form / sending / check-your-email

- [x] **CLI-21** Build: Onboarding flow UI
  - [x] `ConnectPage`, `ConnectForm`, `CheckEmailScreen`
  - [x] Error states: expired token / invalid token / already used
  - [x] Adapted from Variants design (CLI-20)
  - [x] Depends on: CLI-19, CLI-20

- [x] **CLI-22** Design: Portal shell — sidebar + header (Variants)
  - [x] Sidebar: logo, nav items (Dashboard, Pages, Blog, Media, Forms, Settings, Assistant)
  - [x] Header: page title, "View Site →" link, user dropdown
  - [x] Light theme (contrast to agency-hub dark)

- [x] **CLI-23** Build: PortalLayout + sidebar + header
  - [x] `PortalLayout`, `PortalSidebar`, `PortalHeader`
  - [x] Active route highlighting, mobile responsive collapse
  - [x] Branded breadcrumb: tenant site hostname + current page title
  - [x] Depends on: CLI-19, CLI-22

---

## Cycle 4 · M4: Page & Section Editors

- [x] **CLI-24** Design: Pages list + page editor layout (Variants)
  - [x] Pages list table: slug, title, status, last updated
  - [x] Built directly from existing design language (no Variants step)

- [x] **CLI-25** Build: Pages list + page selector
  - [x] `PagesListPage` — loads pages from client Supabase, table with title/slug/status/updated
  - [x] `usePages()` hook — TanStack Query, selects only list columns
  - [x] Loading skeletons, empty state, error state
  - [x] Depends on: CLI-23, CLI-24

- [x] **CLI-26** Build: Page editor + section framework + all section editors
  - [x] `PageEditorPage` — two-column layout: sticky section list + active editor panel
  - [x] `SectionEditor` — dispatcher to known editors, generic fallback for unknown types
  - [x] `SaveIndicator` — idle/saving/saved/error, auto-clears (in components/shared/)
  - [x] Publish/unpublish toggle with ISR trigger
  - [x] Save writes full sections to client Supabase → fires POST /api/revalidate
  - [x] POST /api/revalidate backend endpoint wired

- [x] **CLI-27** Build: Remaining section editors
  - [x] `HeroEditor`, `FeaturesEditor`, `AboutEditor`, `TestimonialsEditor`, `CTAEditor`
  - [x] `editor-primitives.tsx` — shared Field, inputClass, textareaClass
  - [x] FeaturesEditor + TestimonialsEditor support repeatable add/remove rows

---

## Cycle 4 · M5: Blog Editor

- [x] **CLI-28** Design: Blog list + post editor (Variants)
  - [x] Posts table with status badges (Draft / Published)
  - [x] Post editor: Tiptap toolbar + content area + meta sidebar
  - [x] Meta sidebar: slug, excerpt, cover image picker, author, SEO

- [x] **CLI-29** Build: Blog list page
  - [x] `BlogListPage`, `PostsTable`
  - [x] Depends on: CLI-23, CLI-28

- [x] **CLI-30** Build: Post editor (Tiptap)
  - [x] `TiptapEditor` (StarterKit, Image, Link, Placeholder) + toolbar
  - [x] `PostMetaSidebar` (slug, excerpt, author, cover image stub)
  - [x] Publish toggle (published_at set on first publish, not cleared on unpublish)
  - [x] `NewPostPage` (insert on save → redirect to edit), `EditPostPage` (auto-save 2s debounce)
  - [x] Slug: auto-generated from title, editable, unique check on blur
  - [x] Depends on: CLI-29

---

## Cycle 5 · M6: Media Library

- [x] **CLI-31** Design: Media library grid (Variants)
  - [x] Grid of image cards: thumbnail, filename, size, copy URL + delete buttons
  - [x] Upload area: drag-and-drop zone + file picker
  - [x] Media picker modal (for use from other editors)

- [x] **CLI-32** Build: Media library
  - [x] Storage-native: `useStorageFiles`, `useUploadFile`, `useDeleteFile` (no media table)
  - [x] `FileBrowser` — full recursive folder tree with breadcrumb navigation, drag-and-drop
  - [x] `StorageItemCard` — folder or file, copy URL + delete for files
  - [x] Primary bucket derived from `site_url` (e.g. acmecorp-com), auto-created on registration
  - [x] Backend: `createDefaultBucket` called from `RegisterClient` using service role key
  - [x] Depends on: CLI-23, CLI-31

- [x] **CLI-33** Build: MediaPickerModal
  - [x] Radix Dialog wrapping `FileBrowser` in selectable mode — full tree, onSelect closes + returns URL
  - [x] Wired into PostMetaSidebar (cover image)
  - [x] Depends on: CLI-32

---

## Cycle 5 · M7: Claude Assistant UI

- [x] **CLI-34** Design: Assistant panel + diff preview (Variants)
  - [x] Instruction form: page selector, section selector, instruction textarea
  - [x] Diff preview: side-by-side table (field | current | proposed | notes)
  - [x] Apply / Discard action bar
  - [x] Rate limit banner states (minute / hour / budget)

- [x] **CLI-35** Build: Claude assistant backend integration
  - [x] `use-assistant.ts` hook — calls `POST /assistant/generate`
  - [x] Handle 429 responses: map error type to correct user-facing message
  - [x] Depends on: CLI-16, CLI-34

- [x] **CLI-36** Build: Assistant UI components
  - [x] `InstructionForm`, `DiffPreview`, `ApplyBar`, `RateLimitBanner`
  - [x] `AssistantPage` (full-page version at `/assistant`)
  - [x] Apply flow: write changed fields to client Supabase → call portal backend for ISR → show success
  - [x] Depends on: CLI-35, CLI-26

---

## Cycle 6 · M8: Secondary Features

- [x] **CLI-37** Build: Form submissions inbox
  - [x] `FormsPage`, `SubmissionsTable`, `SubmissionDetail` (sheet/drawer)
  - [x] Mark as read on open
  - [x] `007_form_submissions_rls.sql` — RLS policies for anon INSERT/SELECT/UPDATE
  - [x] Depends on: CLI-23

- [x] **CLI-38** Build: Settings pages
  - [x] `SettingsPage` (tabbed), `GeneralSettings`, `SeoSettings`, `SocialSettings`
  - [x] `NavEditor` (drag-to-reorder nav items)
  - [x] `SocialSettings` — dynamic add/remove any platform, stored as JSON in `social_links` key
  - [x] Depends on: CLI-23

- [x] **CLI-39** Build: Dashboard overview
  - [x] `DashboardPage`, `QuickActions`, `RecentEdits`, `FormSubmissionsPreview`
  - [x] `use-recent-edits.ts` — parallel pages + posts queries merged by updated_at
  - [x] Depends on: CLI-23, CLI-37

---

## Cycle 6 · M9: QA & Launch

- [x] **CLI-40** End-to-end: Full onboarding flow (code-traced)
  - [x] connect → rate-limited POST /onboarding/connect → email → GET /onboarding/confirm → JWT → /dashboard
  - [x] Error states: expired/used/invalid token all return correct user-facing copy

- [x] **CLI-41** End-to-end: Content edit → live on site (code-traced)
  - [x] page-editor-page.tsx saves sections to Supabase then POSTs /api/revalidate
  - [x] edit-post-page.tsx auto-saves + fires /api/revalidate with /blog/[slug] + /blog paths
  - [x] revalidate handler fires non-blocking TriggerISR to client site_url

- [x] **CLI-42** End-to-end: Claude assistant → apply (code-traced)
  - [x] assistant-page.tsx: generate → diff preview → apply writes to Supabase → fires /api/revalidate
  - [x] 429 responses mapped: "too quickly" → minute, "Hourly limit" → hour, else → budget

- [x] **CLI-43** Rate limit verification (code-traced)
  - [x] claude ratelimit.go: sliding window 5/min + 20/hour per tenant_id via Redis
  - [x] onboarding: 1/2min per IP; magic-link: 1/2min per IP; auth: 30/min per IP

- [x] **CLI-44** Security review
  - [x] Management token validated on startup — os.Exit(1) if invalid (startup/validate.go)
  - [x] Service role key never in any API response — Profile returns only UserID/TenantID/Email/URL/AnonKey/SiteURL
  - [x] CSRF double-submit cookie enforced on all browser mutations (admin routes correctly exempt)
  - [x] Tenant isolation: JWT tenant_id → tenant.Resolve() → decrypt → client Supabase
  - [x] CORS: explicit allowedOrigin only, never "*", withCredentials required
  - [x] Security headers: nosniff, deny-frame, XSS, referrer, permissions-policy
  - [x] Fixed: api/.env.example had stray UPSTASH_REDIS_TOKEN + missing PUBLIC_URL/MAILER_PROVIDER

- [x] **CLI-45** Deploy to Railway + Vercel
  - [x] Railway: `api/railway.toml` present (nixpacks build, /health check, on_failure restart)
  - [x] Vercel: `web/vercel.json` created (SPA rewrite, dist output)
  - [x] `api/.env.example` updated with correct Upstash URL format + all required vars
  - [x] `web/.env.example` created (VITE_API_BASE_URL)
