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

- [ ] **CLI-10** Implement portal auth (JWT with tenant claims)
  - [ ] `internal/auth/` — full feature
  - [ ] `PortalClaims` struct: `user_id`, `tenant_id`, `email`, `supabase_url`, `supabase_anon_key`
  - [ ] Magic link flow: `POST /auth/magic-link` → Supabase → `/auth/callback` → `POST /auth/exchange` → portal JWT
  - [ ] `GET /auth/profile`, `POST /auth/refresh`, `POST /auth/logout`, `GET /auth/csrf`
  - [ ] Depends on: CLI-9

- [ ] **CLI-11** Implement tenant registry service
  - [ ] `internal/tenant/` — model, repository, service
  - [ ] AES-256-GCM encrypt/decrypt for Supabase credentials (`internal/utils/crypto.go`)
  - [ ] `ResolveTenant()` middleware — inject tenant config into handler context
  - [ ] `POST /api/admin/register-client` endpoint (called by agency-hub, not clients)
  - [ ] Depends on: CLI-10

- [ ] **CLI-12** Implement ISR revalidation service
  - [ ] `internal/revalidate/service.go`
  - [ ] `TriggerISR(siteURL, paths, secret)` — HTTP POST to client-site `/api/revalidate`
  - [ ] Non-blocking: fire-and-forget with error logging
  - [ ] Depends on: CLI-8

---

## Cycle 2 · M2: Claude Content Assistant (Backend)

- [ ] **CLI-13** Implement Claude rate limiter
  - [ ] `internal/claude/ratelimit.go`
  - [ ] Sliding window: 5/min + 20/hour per `tenant_id`
  - [ ] Redis keys: `claude_rl:{tenant_id}:minute`, `claude_rl:{tenant_id}:hour`
  - [ ] Depends on: CLI-8

- [ ] **CLI-14** Implement Claude usage repository
  - [ ] `internal/claude/repository.go`
  - [ ] `RecordUsage()` — calls `POST /api/claude/usage` on agency-hub (fire-and-forget)
  - [ ] `CheckBudget()` — calls `GET /api/claude/budget/{client_id}` on agency-hub
  - [ ] Auth: `Authorization: Bearer AGENCY_MANAGEMENT_TOKEN` + `X-Client-ID` header
  - [ ] Never fail the user request if usage recording fails
  - [ ] Depends on: CLI-11

- [ ] **CLI-15** Implement Claude prompt builder
  - [ ] `internal/claude/prompt.go`
  - [ ] Fetches current section from client's Supabase via service role key
  - [ ] Builds system prompt with: business name, page, section, current content JSON
  - [ ] Returns `(systemPrompt string, currentContent map[string]any, err)`
  - [ ] Depends on: CLI-11

- [ ] **CLI-16** Implement Claude service + handler
  - [ ] `internal/claude/service.go` — orchestrates: rate limit → budget → prompt → API call → record usage
  - [ ] `internal/claude/handler.go` — `POST /assistant/generate`
  - [ ] Parse Claude response as `[]FieldChange` JSON array
  - [ ] Invalid JSON from Claude → return 500 "temporarily unavailable" (not raw parse error)
  - [ ] 429 responses: distinct messages for minute / hour / budget exceeded
  - [ ] Depends on: CLI-13, CLI-14, CLI-15

---

## Cycle 3 · M3: Frontend Shell + Onboarding UI

- [ ] **CLI-17** Set up React + Vite + TypeScript project
  - [ ] Same dependencies as agency-hub `web/`
  - [ ] Add: Supabase JS client, Tiptap + extensions

- [ ] **CLI-18** Set up axios client + Supabase context
  - [ ] `src/lib/axios.ts` — portal backend calls
  - [ ] `src/contexts/supabase-context.tsx` — initialize tenant Supabase client from JWT claims
  - [ ] `useTenantSupabase()` hook

- [ ] **CLI-19** Set up auth context + routes
  - [ ] `auth-context.tsx` with `tenant_id` claim
  - [ ] `routes.tsx` with `ProtectedRoute` + `GuestRoute` guards

- [ ] **CLI-20** Design: Connect (onboarding) page (Variants)
  - [ ] Clean centered layout
  - [ ] Token field + email field
  - [ ] Three states: form / sending / check-your-email

- [ ] **CLI-21** Build: Onboarding flow UI
  - [ ] `ConnectPage`, `ConnectForm`, `CheckEmailScreen`
  - [ ] Error states: expired token / invalid token / already used
  - [ ] Adapted from Variants design (CLI-20)
  - [ ] Depends on: CLI-19, CLI-20

- [ ] **CLI-22** Design: Portal shell — sidebar + header (Variants)
  - [ ] Sidebar: logo, nav items (Dashboard, Pages, Blog, Media, Forms, Settings, Assistant)
  - [ ] Header: page title, "View Site →" link, user dropdown
  - [ ] Light theme (contrast to agency-hub dark)

- [ ] **CLI-23** Build: PortalLayout + sidebar + header
  - [ ] `PortalLayout`, `PortalSidebar`, `PortalHeader`
  - [ ] Active route highlighting, mobile responsive collapse
  - [ ] Adapted from Variants design (CLI-22)
  - [ ] Depends on: CLI-19, CLI-22

---

## Cycle 4 · M4: Page & Section Editors

- [ ] **CLI-24** Design: Pages list + page editor layout (Variants)
  - [ ] Pages list table: slug, title, status, last updated
  - [ ] Page editor: section list on left, active editor on right (or stacked)
  - [ ] Section editor: labeled fields + save button + save indicator
  - [ ] Show 3 section types: Hero, Features, Testimonials

- [ ] **CLI-25** Build: Pages list + page selector
  - [ ] `PagesListPage`, `PagesList`
  - [ ] Depends on: CLI-23, CLI-24

- [ ] **CLI-26** Build: Section editor framework + Hero editor
  - [ ] `PageEditorPage`, `SectionEditor` (type dispatcher), `HeroEditor`
  - [ ] `SaveIndicator` (idle / saving / saved / error — auto-clears after 2s)
  - [ ] `SectionPublishToggle`
  - [ ] Save: write to client Supabase → trigger ISR via portal backend
  - [ ] Adapted from Variants design (CLI-24)
  - [ ] Depends on: CLI-25, CLI-12

- [ ] **CLI-27** Build: Remaining section editors
  - [ ] `FeaturesEditor` (repeatable items), `AboutEditor`, `TestimonialsEditor`, `CTAEditor`
  - [ ] Reuse `SectionEditor` framework from CLI-26
  - [ ] Depends on: CLI-26

---

## Cycle 4 · M5: Blog Editor

- [ ] **CLI-28** Design: Blog list + post editor (Variants)
  - [ ] Posts table with status badges (Draft / Published)
  - [ ] Post editor: Tiptap toolbar + content area + meta sidebar
  - [ ] Meta sidebar: slug, excerpt, cover image picker, author, SEO

- [ ] **CLI-29** Build: Blog list page
  - [ ] `BlogListPage`, `PostsTable`
  - [ ] Depends on: CLI-23, CLI-28

- [ ] **CLI-30** Build: Post editor (Tiptap)
  - [ ] `PostEditor` (Tiptap: StarterKit, Image, Link, Placeholder)
  - [ ] `PostMetaForm` (all meta fields + `MediaPickerModal` for cover image)
  - [ ] `PostStatusToggle` (Draft ↔ Published; `published_at` set on first publish, not cleared on unpublish)
  - [ ] `NewPostPage`, `EditPostPage`
  - [ ] Auto-save: debounce 2s after typing stops
  - [ ] Slug: auto-generated from title, editable, unique check on blur
  - [ ] Adapted from Variants design (CLI-28)
  - [ ] Depends on: CLI-29

---

## Cycle 5 · M6: Media Library

- [ ] **CLI-31** Design: Media library grid (Variants)
  - [ ] Grid of image cards: thumbnail, filename, size, copy URL button
  - [ ] Upload area: drag-and-drop zone + file picker button
  - [ ] Media picker modal (for use from other editors)

- [ ] **CLI-32** Build: Media library
  - [ ] `MediaPage`, `MediaGrid`, `MediaItem`, `MediaUploader`
  - [ ] Direct Supabase Storage upload from browser (anon key + Storage RLS)
  - [ ] Accepts: jpeg, png, webp, gif, svg · Max: 5MB
  - [ ] Copy URL to clipboard button
  - [ ] Adapted from Variants design (CLI-31)
  - [ ] Depends on: CLI-23, CLI-31

- [ ] **CLI-33** Build: MediaPickerModal
  - [ ] Reusable modal: browse library → select → return URL via `onSelect(url)` callback
  - [ ] Used in: `PostMetaForm` (cover image), section editors (images)
  - [ ] Depends on: CLI-32

---

## Cycle 5 · M7: Claude Assistant UI

- [ ] **CLI-34** Design: Assistant panel + diff preview (Variants)
  - [ ] Instruction form: page selector, section selector, instruction textarea
  - [ ] Diff preview: side-by-side table (field | current | proposed | notes)
  - [ ] Apply / Discard action bar
  - [ ] Rate limit banner states (minute / hour / budget)

- [ ] **CLI-35** Build: Claude assistant backend integration
  - [ ] `use-assistant.ts` hook — calls `POST /assistant/generate`
  - [ ] Handle 429 responses: map error type to correct user-facing message
  - [ ] Depends on: CLI-16, CLI-34

- [ ] **CLI-36** Build: Assistant UI components
  - [ ] `InstructionForm`, `DiffPreview`, `ApplyBar`, `RateLimitBanner`
  - [ ] `AssistantPanel` (floating panel, accessible from page/post editors)
  - [ ] `AssistantPage` (full-page version at `/assistant`)
  - [ ] Apply flow: write changed fields to client Supabase → call portal backend for ISR → show success
  - [ ] Adapted from Variants design (CLI-34)
  - [ ] Depends on: CLI-35, CLI-26

---

## Cycle 6 · M8: Secondary Features

- [ ] **CLI-37** Build: Form submissions inbox
  - [ ] `FormsPage`, `SubmissionsTable`, `SubmissionDetail` (sheet/drawer)
  - [ ] Mark as read on open
  - [ ] Depends on: CLI-23

- [ ] **CLI-38** Build: Settings pages
  - [ ] `SettingsPage` (tabbed), `GeneralSettings`, `SeoSettings`, `SocialSettings`
  - [ ] `NavEditor` (drag-to-reorder nav items)
  - [ ] Depends on: CLI-23

- [ ] **CLI-39** Build: Dashboard overview
  - [ ] `DashboardPage`, `QuickActions`, `RecentEdits`, `FormSubmissionsPreview`
  - [ ] Depends on: CLI-23, CLI-37

---

## Cycle 6 · M9: QA & Launch

- [ ] **CLI-40** End-to-end: Full onboarding flow
  - [ ] Admin generates `connection_token` in agency-hub
  - [ ] Client enters token + email on `/connect`
  - [ ] Email arrives → click link → land on `/dashboard`

- [ ] **CLI-41** End-to-end: Content edit → live on site
  - [ ] Edit hero section → save → ISR triggered → site reloads → updated content visible

- [ ] **CLI-42** End-to-end: Claude assistant → apply
  - [ ] Type instruction → see diff → apply → content in Supabase updated → ISR → live

- [ ] **CLI-43** Rate limit verification
  - [ ] Fire 6 requests under a minute — 6th gets 429 with correct minute-limit message
  - [ ] Fire 21 requests in an hour — 21st gets 429 with correct hour-limit message
  - [ ] Exhaust monthly budget (reduced test limit) — correct budget message shown

- [ ] **CLI-44** Security review
  - [ ] Management token validation on startup confirmed
  - [ ] Service role key absent from all API responses
  - [ ] CSRF enforced on all mutations
  - [ ] Tenant isolation verified (tenant A cannot access tenant B's Supabase data)

- [ ] **CLI-45** Deploy to Railway + Vercel
  - [ ] Portal backend → Railway (`VITE_API_BASE_URL` set in env)
  - [ ] Portal frontend → Vercel
  - [ ] Portal URL recorded in agency-hub for all active clients
