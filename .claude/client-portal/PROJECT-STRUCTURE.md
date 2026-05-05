# client-portal — Project Structure

---

## Repository Root

```
client-portal/
├── api/
├── web/
├── .github/workflows/deploy.yml
└── README.md
```

---

## Backend (`api/`)

```
api/
├── cmd/
│   ├── app/main.go                    # Entry: config → DB → Redis → validate management
│   │                                  #   token → wire routes → serve
│   └── migrate/main.go
│
├── internal/
│   │
│   ├── startup/
│   │   └── validate.go                # ValidateManagementToken() — called in main.go
│   │                                  #   before serving. Calls agency-hub API.
│   │                                  #   os.Exit(1) if invalid.
│   │
│   ├── onboarding/
│   │   ├── model.go                   # ConnectRequest {connection_token, email},
│   │   │                              #   ConfirmRequest {token},
│   │   │                              #   EmailConfirmation DB row
│   │   ├── repository.go              # StoreEmailConfirmation, GetByTokenHash,
│   │   │                              #   MarkConfirmationUsed
│   │   ├── service.go                 # Connect(): validate token with agency-hub →
│   │   │                              #   create Supabase Auth user via service role →
│   │   │                              #   send confirmation email
│   │   │                              # Confirm(): verify token → mark used →
│   │   │                              #   issue portal JWT
│   │   ├── handler.go                 # POST /onboarding/connect,
│   │   │                              #   GET /onboarding/confirm?token=...
│   │   └── routes.go
│   │
│   ├── auth/
│   │   ├── model.go                   # LoginRequest, PortalClaims (includes tenant_id,
│   │   │                              #   client_supabase_url, client_supabase_anon_key)
│   │   ├── repository.go              # Refresh token store
│   │   ├── service.go                 # Login (magic link or password), Logout, Refresh,
│   │   │                              #   IssuePortalJWT (embeds tenant Supabase config)
│   │   ├── handler.go                 # POST /auth/login, POST /auth/logout,
│   │   │                              #   POST /auth/refresh, GET /auth/csrf,
│   │   │                              #   GET /auth/profile
│   │   └── routes.go
│   │
│   ├── tenant/
│   │   ├── model.go                   # Tenant, TenantConfig
│   │   ├── repository.go              # GetTenantByID, UpsertTenant,
│   │   │                              #   GetSupabaseConfig (decrypt credentials)
│   │   └── service.go                 # ResolveTenant (from JWT claim) — used by
│   │                                  #   other services to get tenant's Supabase config
│   │
│   ├── claude/
│   │   ├── model.go                   # GenerateRequest, FieldChange, GenerateResponse,
│   │   │                              #   UsageRecord, ClientBudget
│   │   ├── repository.go              # RecordUsage (upsert to agency claude_usage),
│   │   │                              #   GetClientBudget, CheckAndAlertBudget
│   │   ├── ratelimit.go               # CheckAndIncrement (minute + hour sliding window)
│   │   ├── prompt.go                  # PromptBuilder.Build() — fetch section from
│   │   │                              #   client Supabase, compose system prompt
│   │   ├── service.go                 # Generate(): rate limit → budget → call Claude →
│   │   │                              #   parse response → record usage (async)
│   │   ├── handler.go                 # POST /assistant/generate
│   │   └── routes.go
│   │
│   ├── revalidate/
│   │   ├── model.go                   # RevalidateRequest {paths []string}
│   │   └── service.go                 # TriggerISR(clientSiteURL, paths, secret)
│   │                                  #   Called after every confirmed content mutation
│   │                                  #   No handler — called internally by Claude service
│   │                                  #   after Apply and by content mutation confirmations
│   │
│   ├── config/
│   │   └── config.go
│   │
│   ├── database/
│   │   ├── db.go
│   │   └── migrate.go
│   │
│   ├── middleware/
│   │   ├── auth.go                    # Extract + validate portal JWT, inject PortalClaims
│   │   ├── csrf.go
│   │   ├── ratelimit.go               # General rate limiting (not Claude-specific)
│   │   ├── security.go
│   │   └── logger.go
│   │
│   └── utils/
│       ├── crypto.go                  # Encrypt/decrypt tenant Supabase credentials
│       ├── response.go
│       └── errors.go
│
├── pkg/logger/logger.go
├── docs/
├── supabase/
│   └── migrations/
│       ├── 001_create_tenants.sql
│       └── 002_create_email_confirmations.sql
├── .env.example
├── .air.toml
├── railway.toml
└── go.mod
```

---

## Frontend (`web/`)

```
web/src/
│
├── components/
│   ├── layout/
│   │   ├── PortalLayout.tsx           # Authenticated shell: sidebar + content
│   │   ├── PortalSidebar.tsx          # Nav: Dashboard, Pages, Blog, Media,
│   │   │                              #   Forms, Settings, Assistant
│   │   └── PortalHeader.tsx           # Page title + site link + user menu
│   │
│   ├── guards/
│   │   ├── ProtectedRoute.tsx
│   │   └── GuestRoute.tsx
│   │
│   ├── shared/
│   │   ├── SaveIndicator.tsx          # "Saving..." / "Saved" / "Error" — used in editors
│   │   ├── EmptyState.tsx             # Reusable empty state with icon + message + CTA
│   │   └── ConfirmDialog.tsx          # Generic "Are you sure?" dialog (shadcn AlertDialog)
│   │
│   └── ui/                            # shadcn/ui components
│       # Required: button, card, input, label, badge, table, dialog, sheet,
│       # dropdown-menu, avatar, skeleton, separator, alert, progress,
│       # tabs, popover, textarea, form, switch, scroll-area, alert-dialog
│
├── config/routes.tsx
│
├── contexts/
│   ├── auth-context.tsx               # Portal auth state (user, tenant_id, isLoading)
│   └── supabase-context.tsx           # Tenant Supabase client
│                                      #   Initialized once after login using
│                                      #   URL + anon key from JWT
│                                      #   Hook: useTenantSupabase()
│
├── features/
│   │
│   ├── onboarding/
│   │   ├── pages/
│   │   │   └── ConnectPage.tsx        # Step 1: enter connection_token + email
│   │   │                              #   Shows: form → sending → check email
│   │   ├── components/
│   │   │   ├── ConnectForm.tsx        # Token + email fields, error states
│   │   │   │                          #   Errors: expired token, invalid token,
│   │   │   │                          #   already used, network error
│   │   │   └── CheckEmailScreen.tsx   # "Check your inbox" confirmation screen
│   │   ├── hooks/
│   │   │   └── use-onboarding.ts      # useConnect() mutation
│   │   └── services/
│   │       └── onboarding.service.ts  # connect(token, email), confirm(token)
│   │
│   ├── dashboard/
│   │   ├── pages/
│   │   │   └── DashboardPage.tsx      # Portal overview
│   │   └── components/
│   │       ├── QuickActions.tsx       # Shortcut cards: New Post, Edit Home, Upload Media
│   │       ├── RecentEdits.tsx        # Last 5 content changes (from Supabase updated_at)
│   │       └── FormSubmissionsPreview.tsx # Unread submissions count + last few
│   │
│   ├── pages/
│   │   ├── pages/
│   │   │   ├── PagesListPage.tsx      # All pages with slug, title, publish status
│   │   │   └── PageEditorPage.tsx     # Section-by-section editor for one page
│   │   ├── components/
│   │   │   ├── PagesList.tsx          # Table: slug, title, status, last updated, edit btn
│   │   │   ├── SectionEditor.tsx      # Renders the right editor for each section type:
│   │   │   │                          #   HeroEditor, FeaturesEditor, AboutEditor, etc.
│   │   │   ├── HeroEditor.tsx         # Fields: headline, subheadline, cta_label, cta_url
│   │   │   ├── FeaturesEditor.tsx     # Repeatable list of feature items
│   │   │   ├── AboutEditor.tsx        # Textarea for body, optional image
│   │   │   ├── TestimonialsEditor.tsx # Repeatable testimonial items
│   │   │   └── SectionPublishToggle.tsx # is_published switch per page
│   │   ├── hooks/
│   │   │   └── use-pages.ts           # usePages(), usePage(slug), useUpdateSection()
│   │   └── services/
│   │       └── pages.service.ts       # All queries use useTenantSupabase()
│   │
│   ├── blog/
│   │   ├── pages/
│   │   │   ├── BlogListPage.tsx       # All posts: title, slug, status, published_at
│   │   │   ├── NewPostPage.tsx        # Create post
│   │   │   └── EditPostPage.tsx       # Edit post
│   │   ├── components/
│   │   │   ├── PostsTable.tsx         # Sortable table with status badges
│   │   │   ├── PostEditor.tsx         # Tiptap rich text editor
│   │   │   │                          #   Toolbar: bold, italic, h2/h3, lists,
│   │   │   │                          #   blockquote, link, image (uploads to Storage)
│   │   │   ├── PostMetaForm.tsx       # Slug, excerpt, cover image, author, SEO fields
│   │   │   └── PostStatusToggle.tsx   # Draft | Published toggle
│   │   ├── hooks/
│   │   │   └── use-posts.ts           # usePosts(), usePost(id), useCreatePost(),
│   │   │                              #   useUpdatePost(), useDeletePost()
│   │   └── services/
│   │       └── posts.service.ts
│   │
│   ├── media/
│   │   ├── pages/
│   │   │   └── MediaPage.tsx          # Media library grid
│   │   ├── components/
│   │   │   ├── MediaGrid.tsx          # Image grid with filename + size
│   │   │   ├── MediaUploader.tsx      # Drag-and-drop + file picker
│   │   │   │                          #   Uploads to Supabase Storage 'media' bucket
│   │   │   ├── MediaItem.tsx          # Individual media card with copy URL + delete
│   │   │   └── MediaPickerModal.tsx   # Reusable: pick an image from library
│   │   │                              #   Used by: PostEditor, SectionEditors
│   │   ├── hooks/
│   │   │   └── use-media.ts           # useMedia(), useUploadMedia(), useDeleteMedia()
│   │   └── services/
│   │       └── media.service.ts       # Direct Supabase Storage operations
│   │
│   ├── forms/
│   │   ├── pages/
│   │   │   └── FormsPage.tsx          # Inbox for form_submissions
│   │   └── components/
│   │       ├── SubmissionsTable.tsx   # Date, form name, data preview, read status
│   │       └── SubmissionDetail.tsx   # Full data JSONB rendered in a sheet
│   │
│   ├── settings/
│   │   ├── pages/
│   │   │   └── SettingsPage.tsx       # Tabbed: General | SEO | Social | Navigation
│   │   └── components/
│   │       ├── GeneralSettings.tsx    # site_name, tagline, logo, contact info
│   │       ├── SeoSettings.tsx        # seo_title, seo_description, og_image
│   │       ├── SocialSettings.tsx     # Social link fields
│   │       └── NavEditor.tsx          # Reorderable nav items (drag to reorder)
│   │
│   └── assistant/
│       ├── pages/
│       │   └── AssistantPage.tsx      # Full-page assistant (also accessible as panel)
│       └── components/
│           ├── AssistantPanel.tsx     # Floating panel — accessible from any editor page
│           │                          #   Input: page/section selector + instruction textarea
│           │                          #   States: idle → generating → preview → applied/discarded
│           ├── InstructionForm.tsx    # Page selector, section selector, instruction textarea
│           ├── DiffPreview.tsx        # Side-by-side: Current | Proposed
│           │                          #   Each FieldChange as a row with field name + values
│           ├── ApplyBar.tsx           # "Apply Changes" + "Discard" buttons
│           │                          #   Shows token count + "powered by Claude"
│           └── RateLimitBanner.tsx    # Shown when 429 — clear message for each limit type
│
├── lib/
│   ├── axios.ts                       # Portal backend calls (onboarding, Claude proxy)
│   ├── supabase.ts                    # createClient factory (initialized from context)
│   └── utils.ts                       # cn(), formatDate(), formatBytes()
│
├── types/
│   └── index.ts                       # Shared: Page, Post, NavItem, SiteSetting,
│                                      #   FormSubmission, MediaItem, FieldChange
│
└── utils/
    ├── errors.ts
    └── content.ts                     # jsonbToSections(), sectionToDisplay()
```

---

## Route Map

| Path | Component | Guard |
|------|-----------|-------|
| `/connect` | `ConnectPage` | GuestRoute |
| `/confirm` | (handled by backend redirect) | — |
| `/dashboard` | `DashboardPage` | ProtectedRoute |
| `/pages` | `PagesListPage` | ProtectedRoute |
| `/pages/:slug` | `PageEditorPage` | ProtectedRoute |
| `/blog` | `BlogListPage` | ProtectedRoute |
| `/blog/new` | `NewPostPage` | ProtectedRoute |
| `/blog/:id/edit` | `EditPostPage` | ProtectedRoute |
| `/media` | `MediaPage` | ProtectedRoute |
| `/forms` | `FormsPage` | ProtectedRoute |
| `/settings` | `SettingsPage` | ProtectedRoute |
| `/assistant` | `AssistantPage` | ProtectedRoute |

---

## Component Build Priority

```
Phase 1 — Shell + Onboarding (the door in)
  ConnectForm, ConnectPage, CheckEmailScreen
  PortalLayout, PortalSidebar, PortalHeader

Phase 2 — Core content editors
  PagesList, PageEditorPage, SectionEditor + all section editors (Hero, Features, About)
  SaveIndicator (shared)

Phase 3 — Blog
  PostsTable, PostEditor (Tiptap), PostMetaForm, PostStatusToggle

Phase 4 — Media
  MediaGrid, MediaUploader, MediaItem, MediaPickerModal

Phase 5 — Assistant
  InstructionForm, DiffPreview, ApplyBar, RateLimitBanner, AssistantPanel

Phase 6 — Secondary
  FormsPage, SubmissionsTable, SettingsPage + all settings components

Phase 7 — Dashboard
  QuickActions, RecentEdits, FormSubmissionsPreview
```

---

## Supabase Migrations (Portal's Own DB)

```sql
-- 001_create_tenants.sql
CREATE TABLE tenants (
    id                        UUID PRIMARY KEY,  -- same as client_id from agency-hub
    supabase_url_encrypted    TEXT NOT NULL,
    supabase_anon_encrypted   TEXT NOT NULL,
    onboarded_at              TIMESTAMPTZ,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 002_create_email_confirmations.sql
CREATE TABLE email_confirmations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    email        TEXT NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    used_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```
