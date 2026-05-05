# client-portal — Linear Project Setup

---

## Workspace Setup

```
Team name:       client-portal
Team identifier: CLI   (issues: CLI-5, CLI-6, ...)
```

Same labels as agency-hub: `backend` `frontend` `design` `infra` `security` `bug` `chore`

---

## Milestones

```
M1: Backend Foundation (startup validation, onboarding, auth)
M2: Claude Content Assistant (backend)
M3: Frontend Shell + Onboarding UI
M4: Page & Section Editors
M5: Blog Editor
M6: Media Library
M7: Claude Assistant UI
M8: Secondary Features (Forms, Settings, Dashboard)
M9: QA & Launch
```

---

## Cycles

```
Cycle 1: M1 — Backend foundation
Cycle 2: M2 — Claude backend
Cycle 3: M3 — Frontend shell + onboarding
Cycle 4: M4 + M5 — Content editors
Cycle 5: M6 + M7 — Media + assistant
Cycle 6: M8 + M9 — Secondary + QA
```

---

## All Issues

### M1 — Backend Foundation

**CLI-5** Set up Go project + install dependencies
`backend` `chore` · Urgent · M1
- Chi v5, validator, swag, air, anthropic-sdk-go
- Create all internal/ feature folders

**CLI-6** Set up portal's own Supabase project
`infra` · Urgent · M1
- Create Supabase project: client-portal-prod
- Save credentials to 1Password
- Run migrations (tenants + email_confirmations tables)

**CLI-7** Implement startup management token validation
`backend` `security` · Urgent · M1
- internal/startup/validate.go
- Calls GET /validate-management-token on agency-hub at startup
- os.Exit(1) if invalid — fail loud, not silent

**CLI-8** Implement config, DB, middleware stack
`backend` `chore` · Urgent · M1
- config.go, db.go, security.go, logger.go, csrf.go, ratelimit.go, auth.go

**CLI-9** Implement onboarding flow (backend)
`backend` `security` · Urgent · M1
- internal/onboarding/ — full feature
- POST /onboarding/connect:
  1. Call agency-hub POST /validate-connection-token with entered token
  2. If valid: store email confirmation in portal DB (do NOT create Supabase user yet)
  3. Send confirmation email via Resend
  Rate limit: 5 req/min per IP
- GET /onboarding/confirm?token=...:
  1. Verify + consume confirmation token from portal DB
  2. Create user in client's Supabase Auth via service role key
  3. Mark tenant onboarded in portal tenant registry
  4. Create tenant_users record (email → tenant_id mapping for future logins)
  5. Issue portal JWT with tenant Supabase config embedded
- Depends on: CLI-7, CLI-8

**CLI-10** Implement portal auth (JWT with tenant claims)
`backend` `security` · Urgent · M1
- internal/auth/ — full feature
- PortalClaims struct includes: user_id, tenant_id, client_supabase_url, client_supabase_anon_key
- GET /auth/profile, POST /auth/refresh, POST /auth/logout, GET /auth/csrf
- Depends on: CLI-9

**CLI-11** Implement tenant registry service
`backend` · High · M1
- internal/tenant/ — model, repository, service
- Encrypt/decrypt Supabase config stored in tenants table
- ResolveTenant() middleware — inject tenant config into handler context
- Depends on: CLI-10

**CLI-12** Implement ISR revalidation service
`backend` · High · M1
- internal/revalidate/service.go
- TriggerISR(siteURL, paths, secret) — HTTP POST to client-site
- Called after every confirmed content update
- Non-blocking: fire-and-forget with error logging
- Depends on: CLI-8

---

### M2 — Claude Content Assistant (Backend)

**CLI-13** Implement Claude rate limiter
`backend` `security` · Urgent · M2
- internal/claude/ratelimit.go
- Sliding window: 5/min + 20/hour per tenant_id
- Redis key pattern: `claude_rl:{tenant_id}:minute`, `claude_rl:{tenant_id}:hour`
- Depends on: CLI-8

**CLI-14** Implement Claude usage repository
`backend` · Urgent · M2
- internal/claude/repository.go
- RecordUsage — calls POST /api/claude/usage on agency-hub (fire-and-forget, never fail user request)
- CheckBudget — calls GET /api/claude/budget/{client_id} on agency-hub
- Auth on both: Authorization: Bearer AGENCY_MANAGEMENT_TOKEN + X-Client-ID header
- Depends on: CLI-11

**CLI-15** Implement Claude prompt builder
`backend` · Urgent · M2
- internal/claude/prompt.go
- Fetches current section from client's Supabase via service role key
- Builds system prompt with: business name, page, section, current content JSON
- Returns (systemPrompt string, currentContent map[string]any, err)
- Depends on: CLI-11

**CLI-16** Implement Claude service + handler
`backend` · Urgent · M2
- internal/claude/service.go — orchestrates: rate limit → budget → prompt → API call → record usage
- internal/claude/handler.go — POST /assistant/generate
- Parse Claude response as []FieldChange JSON array
- 429 responses: distinct messages for minute limit, hour limit, budget exceeded
- Depends on: CLI-13, CLI-14, CLI-15

---

### M3 — Frontend Shell + Onboarding UI

**CLI-17** Set up React + Vite + TypeScript project
`frontend` `chore` · Urgent · M3
- Same dependencies as agency-hub web/
- Add: Supabase JS client, Tiptap and extensions

**CLI-18** Set up axios client + Supabase context
`frontend` · Urgent · M3
- src/lib/axios.ts (portal backend calls)
- src/contexts/supabase-context.tsx — initialize Supabase client from JWT claims
- useTenantSupabase() hook

**CLI-19** Set up auth context + routes
`frontend` · Urgent · M3
- auth-context.tsx with tenant_id claim
- Central routes.tsx with ProtectedRoute + GuestRoute

**CLI-20** Design: Connect (onboarding) page (Variants)
`design` · High · M3
- Clean centered layout
- Token field + email field
- Three states: form, sending, check-your-email

**CLI-21** Build: Onboarding flow UI
`frontend` · High · M3
- ConnectPage, ConnectForm, CheckEmailScreen
- Error states: expired token, invalid token, already used
- Adapted from Variants design (CLI-20)
- Depends on: CLI-19, CLI-20

**CLI-22** Design: Portal shell — sidebar + header (Variants)
`design` · High · M3
- Sidebar: logo, nav items (Dashboard, Pages, Blog, Media, Forms, Settings, Assistant)
- Header: page title, "View Site →" link, user dropdown
- Light theme (contrast to agency-hub dark)

**CLI-23** Build: PortalLayout + sidebar + header
`frontend` · High · M3
- PortalLayout, PortalSidebar, PortalHeader
- Active route highlighting, mobile responsive collapse
- Adapted from Variants design (CLI-22)
- Depends on: CLI-19, CLI-22

---

### M4 — Page & Section Editors

**CLI-24** Design: Pages list + page editor layout (Variants)
`design` · High · M4
- Pages list table: slug, title, status, last updated
- Page editor: section list on left, active editor on right (or stacked)
- Section editor: labeled fields with save button + save indicator
- Show 3 section types: Hero, Features, Testimonials

**CLI-25** Build: Pages list + page selector
`frontend` · High · M4
- PagesListPage, PagesList
- Depends on: CLI-23, CLI-24

**CLI-26** Build: Section editor framework + Hero editor
`frontend` · Urgent · M4
- PageEditorPage, SectionEditor (dispatcher), HeroEditor
- SaveIndicator (write to Supabase → success → trigger ISR via backend)
- SectionPublishToggle
- Adapted from Variants design (CLI-24)
- Depends on: CLI-25, CLI-12

**CLI-27** Build: Remaining section editors
`frontend` · High · M4
- FeaturesEditor (repeatable items), AboutEditor, TestimonialsEditor
- Reuse SectionEditor framework from CLI-26
- Depends on: CLI-26

---

### M5 — Blog Editor

**CLI-28** Design: Blog list + post editor (Variants)
`design` · High · M5
- Posts table with status badges
- Post editor: Tiptap toolbar + content area + meta sidebar
- Meta sidebar: slug, excerpt, cover image picker, author, SEO

**CLI-29** Build: Blog list page
`frontend` · High · M5
- BlogListPage, PostsTable
- Depends on: CLI-23, CLI-28

**CLI-30** Build: Post editor (Tiptap)
`frontend` · Urgent · M5
- PostEditor (Tiptap with extensions: StarterKit, Image, Link, Placeholder)
- PostMetaForm (all meta fields + MediaPickerModal for cover image)
- PostStatusToggle (Draft ↔ Published)
- NewPostPage, EditPostPage
- Adapted from Variants design (CLI-28)
- Depends on: CLI-29

---

### M6 — Media Library

**CLI-31** Design: Media library grid (Variants)
`design` · Medium · M6
- Grid of image cards: thumbnail, filename, size, copy URL button
- Upload area: drag-and-drop zone + file picker button
- Media picker modal (for use from editors)

**CLI-32** Build: Media library
`frontend` · High · M6
- MediaPage, MediaGrid, MediaItem, MediaUploader
- Direct Supabase Storage upload from browser (anon key + Storage RLS)
- Copy URL to clipboard button
- Adapted from Variants design (CLI-31)
- Depends on: CLI-23, CLI-31

**CLI-33** Build: MediaPickerModal
`frontend` · High · M6
- Reusable modal: browse library → select → return URL to caller
- Used in: PostMetaForm (cover image), section editors (images)
- Depends on: CLI-32

---

### M7 — Claude Assistant UI

**CLI-34** Design: Assistant panel + diff preview (Variants)
`design` · High · M7
- Instruction form: page selector, section selector, instruction textarea
- Diff preview: side-by-side table (field | current | proposed)
- Apply/Discard action bar
- Rate limit banner states (minute, hour, budget)

**CLI-35** Build: Claude assistant backend integration
`frontend` · High · M7
- use-assistant.ts hook (calls POST /assistant/generate)
- Handle 429 responses: map error type to correct user message
- Depends on: CLI-16, CLI-34

**CLI-36** Build: Assistant UI components
`frontend` · High · M7
- InstructionForm, DiffPreview, ApplyBar, RateLimitBanner
- AssistantPanel (floating panel accessible from page/post editors)
- AssistantPage (full-page version)
- Apply flow: write to Supabase → call portal backend for ISR trigger → show success
- Adapted from Variants design (CLI-34)
- Depends on: CLI-35, CLI-26

---

### M8 — Secondary Features

**CLI-37** Build: Form submissions inbox
`frontend` · Medium · M8
- FormsPage, SubmissionsTable, SubmissionDetail (sheet)
- Mark as read on open
- Depends on: CLI-23

**CLI-38** Build: Settings pages
`frontend` · Medium · M8
- SettingsPage (tabbed), GeneralSettings, SeoSettings, SocialSettings
- NavEditor (drag-to-reorder nav items)
- Depends on: CLI-23

**CLI-39** Build: Dashboard overview
`frontend` · Medium · M8
- DashboardPage, QuickActions, RecentEdits, FormSubmissionsPreview
- Depends on: CLI-23, CLI-37

---

### M9 — QA & Launch

**CLI-40** End-to-end: Full onboarding flow
`backend` `frontend` · Urgent · M9
- Admin generates connection_token in agency-hub
- Client enters token + email on /connect
- Email arrives, click link, land on dashboard

**CLI-41** End-to-end: Content edit → live on site
`backend` `frontend` · Urgent · M9
- Edit hero section → save → ISR triggered → site reloads → updated content visible

**CLI-42** End-to-end: Claude assistant → apply
`backend` `frontend` · Urgent · M9
- Type instruction → see diff → apply → content in Supabase updated → ISR → live

**CLI-43** Rate limit verification
`backend` · High · M9
- Fire 6 requests in under a minute — 6th gets 429 with correct message
- Fire 21 requests in an hour — 21st gets 429 with correct message
- Exhaust monthly token budget (test with reduced limit) — correct budget message shown

**CLI-44** Security review
`security` · High · M9
- Management token validation on startup confirmed
- Service role key not in any API response
- CSRF enforced on all mutations
- Tenant isolation verified (tenant A cannot access tenant B's Supabase data)

**CLI-45** Deploy to Railway + Vercel
`infra` · Urgent · M9
- Portal backend to Railway
- Portal frontend to Vercel (VITE_API_BASE_URL set)
- Portal URL recorded in agency-hub for all active clients
