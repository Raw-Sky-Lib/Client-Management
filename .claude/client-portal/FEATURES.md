# client-portal — Stage-by-Stage Feature Specification
> Reference at the start of every coding session for this project.
> Covers every feature, validation rule, edge case, and acceptance criterion.

---

## Stage 1 — Foundation + Startup Validation

### Go Setup

Same dependency set as agency-hub plus:
```bash
go get github.com/anthropics/anthropic-sdk-go
go get github.com/supabase-community/supabase-go  # for service-role writes to client Supabase
```

### Startup: Management Token Validation (`internal/startup/validate.go`)

This runs **before** the HTTP server starts. If it fails, `os.Exit(1)`.

```go
func ValidateManagementToken(cfg *config.Config, httpClient *http.Client) error {
    req, _ := http.NewRequest("GET", cfg.AgencyAPIURL+"/api/validate-management-token", nil)
    req.Header.Set("Authorization", "Bearer "+cfg.AgencyManagementToken)
    req.Header.Set("X-Client-ID", cfg.AgencyClientID)

    resp, err := httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("could not reach agency-hub: %w", err)
    }
    if resp.StatusCode == 401 {
        return fmt.Errorf("management token is invalid or revoked — update AGENCY_MANAGEMENT_TOKEN env var")
    }
    if resp.StatusCode == 403 {
        return fmt.Errorf("client is not active in agency-hub — check client status")
    }
    if resp.StatusCode != 200 {
        return fmt.Errorf("unexpected status from agency-hub: %d", resp.StatusCode)
    }
    return nil
}
```

Called in `main.go`:
```go
slog.Info("validating management token with agency-hub...")
if err := startup.ValidateManagementToken(cfg, &http.Client{Timeout: 10 * time.Second}); err != nil {
    slog.Error("startup validation failed", "error", err)
    os.Exit(1)
}
slog.Info("management token validated — starting server")
```

**Retry logic:** Retry 3 times with 2-second backoff before giving up. Useful during Railway cold start when agency-hub might not be fully up yet.

### Config

Additional fields beyond agency-hub:
```go
type Config struct {
    // ... (standard fields from agency-hub)
    AgencyAPIURL           string // AGENCY_API_URL
    AgencyClientID         string // AGENCY_CLIENT_ID
    AgencyManagementToken  string // AGENCY_MANAGEMENT_TOKEN
    AnthropicAPIKey        string // ANTHROPIC_API_KEY
    AnthropicDefaultModel  string // ANTHROPIC_DEFAULT_MODEL (default: claude-haiku-4-5-20251001)
    ClaudeDefaultBudget    int    // CLAUDE_DEFAULT_MONTHLY_TOKEN_BUDGET (default: 150000)
    // Portal's own Supabase (tenant registry only)
    SupabaseDBURL          string // SUPABASE_DB_URL
}
```

Required vars: `AGENCY_API_URL`, `AGENCY_CLIENT_ID`, `AGENCY_MANAGEMENT_TOKEN`, `ANTHROPIC_API_KEY`, `JWT_SECRET`, `SUPABASE_DB_URL`, `RESEND_API_KEY`, `UPSTASH_REDIS_URL`, `UPSTASH_REDIS_TOKEN`

### Portal's Own DB Migrations

```sql
-- 001_create_tenants.sql
CREATE TABLE tenants (
    id                      UUID PRIMARY KEY,       -- same as client_id from agency-hub
    supabase_url_encrypted  TEXT NOT NULL,
    supabase_anon_encrypted TEXT NOT NULL,
    onboarded_at            TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
```

### Credential Encryption (`internal/utils/crypto.go`)

Client Supabase credentials are sensitive. Encrypt before storing in portal DB.

- Algorithm: AES-256-GCM
- Key: derived from `JWT_SECRET` via HKDF-SHA256 to 32 bytes
- `EncryptString(plaintext, key string) (ciphertext string, err error)` — returns base64-encoded
- `DecryptString(ciphertext, key string) (plaintext string, err error)`

This means even if the portal DB is compromised, service role keys are not in plaintext.

### Acceptance Criteria — Stage 1
- [ ] App starts and prints "management token validated" log line
- [ ] With invalid management token: app exits with clear error before serving requests
- [ ] Migrations run clean
- [ ] EncryptString + DecryptString round-trip correctly

---

## Stage 2 — Onboarding Flow

### Register Tenant (`POST /api/admin/register-client`)

This endpoint is called by agency-hub (not by clients) when a new client is registered in the portal system. Auth: `Authorization: Bearer <AGENCY_MANAGEMENT_TOKEN>` + `X-Client-ID`.

```go
type RegisterClientRequest struct {
    ClientID              string `json:"client_id"               validate:"required,uuid"`
    ClientSupabaseURL     string `json:"client_supabase_url"     validate:"required,url"`
    ClientSupabaseAnonKey string `json:"client_supabase_anon_key" validate:"required"`
}
```

Flow:
1. Validate management token (same as startup validation)
2. Encrypt `ClientSupabaseURL` and `ClientSupabaseAnonKey`
3. UPSERT into `tenants` (in case of re-registration)
4. Return `201 { "registered": true }`

No CSRF on this endpoint — API-to-API call.

### Connect (`POST /api/onboarding/connect`)

Called when a client user first enters their connection token on `/connect`.

**Rate limit: 5 req/min per IP** — prevent brute-forcing tokens

**Request:**
```go
type ConnectRequest struct {
    ConnectionToken string `json:"connection_token" validate:"required,min=64,max=64"`
    Email           string `json:"email"            validate:"required,email"`
}
```

**Flow:**
1. Validate request
2. Call agency-hub `POST /api/validate-connection-token`:
   - Header: `Authorization: Bearer <AGENCY_MANAGEMENT_TOKEN>`, `X-Client-ID: <AGENCY_CLIENT_ID>`
   - Body: `{ "token": token, "email": email }`
   - If `valid: false` + reason `"expired"` → return `400 { "error": "Your access code has expired. Ask your website team for a new one." }`
   - If `valid: false` + reason `"used"` → return `400 { "error": "This access code has already been used. Contact your website team." }`
   - If `valid: false` + reason `"invalid"` → return `400 { "error": "Invalid access code. Check for typos and try again." }`
3. Agency-hub returns `{ "client_id": "...", "client_name": "..." }`
4. Fetch tenant record from portal DB using `client_id` — if not registered, return `400 { "error": "This client is not set up in the portal yet. Contact your website team." }`
5. Generate confirmation token: `crypto/rand` 32 bytes → hex
6. Store in `email_confirmations`: `{ tenant_id, email, token_hash, expires_at: NOW() + 24h }`
7. Send confirmation email via Resend:
   - Subject: "Confirm your email to access your dashboard"
   - Body: "Click this link to confirm: {FRONTEND_URL}/confirm?token={plaintext_token}"
   - Link expires in 24 hours
8. Return `200 { "message": "Check your email for a confirmation link." }`

**Note:** Do NOT create the Supabase Auth user yet — wait for email confirmation.

### Confirm (`GET /api/onboarding/confirm`)

Called when client clicks the email link. Query param: `?token=<plaintext_token>`

**Flow:**
1. Hash the token
2. Look up in `email_confirmations` by hash
3. If not found → return `400 { "error": "Invalid or expired confirmation link." }`
4. If `used_at IS NOT NULL` → return `400 { "error": "This confirmation link has already been used." }`
5. If `expires_at < NOW()` → return `400 { "error": "Confirmation link expired. Request a new one." }`
6. Fetch tenant record (using `tenant_id` from confirmation row)
7. Decrypt tenant's Supabase URL + anon key from portal DB
8. Create user in client's Supabase Auth via service role key:
   - Use Supabase Admin API: `POST {supabase_url}/auth/v1/admin/users`
   - Headers: `Authorization: Bearer <service_role_key>`, `apikey: <service_role_key>`
   - Body: `{ "email": email, "password": generate_random_password(), "email_confirm": true }`
   - Store the generated password securely (user will login with magic link, not password)
9. Mark `email_confirmations.used_at = NOW()`
10. Mark `tenants.onboarded_at = NOW()` if not already set
11. Issue portal JWT (see Auth section) — embed tenant Supabase config in claims
12. Set JWT cookies
13. Redirect to `{FRONTEND_URL}/dashboard`

**Service role key access:** The service role key is NOT stored in the portal DB (too sensitive). It is fetched at onboarding time from... where?

**Solution:** At onboarding time, the portal calls agency-hub with the management token to get the service role key for this specific onboarding action only:
- `POST /api/clients/onboarding-credentials` on agency-hub (new endpoint)
- Auth: management token
- Returns: `{ "supabase_service_role_key": "..." }` — single-use, logged
- agency-hub stores this key in the clients table (encrypted at rest via Supabase column encryption)
- The key is returned ONLY when creating a Supabase Auth user (once per onboarding)
- After use, it is not stored in portal

**Alternative simpler approach (recommended for v1):** Store the service role key encrypted in the tenants table alongside the anon key. Accept the security tradeoff for simplicity. Document it and revisit in v2 with a secrets manager.

Use the simpler approach for v1. Add `supabase_service_role_encrypted TEXT` to tenants table in a migration. Agency-hub sends it in `register-client` request. Decrypt only when needed for Supabase Auth operations.

### Acceptance Criteria — Stage 2
- [ ] Connect with valid unused token → confirmation email received
- [ ] Connect with expired token → "Your access code has expired" message
- [ ] Connect with already-used token → "already been used" message
- [ ] Confirm link → user created in client Supabase Auth, redirected to dashboard
- [ ] Confirm same link twice → "already been used" error
- [ ] Expired confirmation link → error (set expires_at to past in test)
- [ ] Client not registered in portal → "not set up" message

---

## Stage 3 — Portal Auth

### JWT Claims for Portal

```go
type PortalClaims struct {
    UserID                  string `json:"user_id"`
    TenantID                string `json:"tenant_id"`     // client_id from agency-hub
    Email                   string `json:"email"`
    ClientSupabaseURL       string `json:"supabase_url"`
    ClientSupabaseAnonKey   string `json:"supabase_anon_key"`
    jwt.RegisteredClaims
}
```

**Security note:** The `supabase_anon_key` in the JWT is the public anon key (safe to be in a token the browser can read). Never put service role key in JWT. The JWT itself is in an HTTP-only cookie so the browser can't read it directly — but treat anon key as non-secret.

### Login Endpoints

Portal users (clients) authenticate against their own Supabase Auth project, not the portal's DB.

**Magic Link Login (recommended for clients):**
- `POST /api/auth/magic-link`
- Body: `{ "email": "..." }`
- Portal backend calls Supabase Auth: `POST {supabase_url}/auth/v1/magiclink`
- User clicks link in email → Supabase redirects to `{FRONTEND_URL}/auth/callback?access_token=...`
- Frontend receives Supabase access token → sends to portal backend: `POST /api/auth/exchange`
- Portal backend verifies the Supabase token, looks up tenant by email match, issues portal JWT

**Exchange (`POST /api/auth/exchange`):**
- Body: `{ "supabase_access_token": "..." }`
- Verify Supabase token: `GET {supabase_url}/auth/v1/user` with the token
- Get user.email from Supabase response
- Look up tenant in portal DB that contains this email in their Supabase (tricky: we need to match email to tenant_id)
  - Simpler: store `email → tenant_id` mapping in portal DB when onboarding confirms
  - Add `tenant_users(id, tenant_id, email, created_at)` table
- Issue portal JWT with tenant config embedded
- Set cookies
- Return `200 { "user": { email, tenant_id } }`

**Add migration:** `003_create_tenant_users.sql`
```sql
CREATE TABLE tenant_users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);
```

Create this record at the end of `POST /api/onboarding/confirm`.

**Logout (`POST /api/auth/logout`):** Clear cookies, delete refresh token from Redis.

**Refresh (`POST /api/auth/refresh`):** Same pattern as agency-hub but with PortalClaims.

**CSRF (`GET /api/auth/csrf`):** Same as agency-hub.

### Acceptance Criteria — Stage 3
- [ ] Magic link → click → exchange → portal JWT cookie set
- [ ] Portal JWT contains `tenant_id`, `supabase_url`, `supabase_anon_key`
- [ ] Refresh token rotation works
- [ ] After logout: cookies cleared, protected routes redirect to /connect

---

## Stage 4 — Claude Content Assistant

### Rate Limiter (`internal/claude/ratelimit.go`)

Sliding window per `tenant_id`:

```go
type RateLimiter struct{ rdb *redis.Client }

// Returns nil if within limits, error describing which limit was hit
func (rl *RateLimiter) Check(ctx context.Context, tenantID string) error {
    type limit struct{ window time.Duration; max int; label string }
    limits := []limit{
        {60 * time.Second, 5, "minute"},
        {3600 * time.Second, 20, "hour"},
    }
    for _, l := range limits {
        key := fmt.Sprintf("claude_rl:%s:%s", tenantID, l.label)
        // ZREMRANGEBYSCORE, ZCARD, ZADD, EXPIRE pipeline
        // Return fmt.Errorf("ratelimit_%s", l.label) if at limit
    }
    return nil
}
```

### Usage Repository (`internal/claude/repository.go`)

This writes to the **agency Supabase** (not portal DB) via direct PostgreSQL connection string (the agency DB URL is a separate env var or accessed via agency-hub API).

**Option A (recommended for v1):** The portal backend calls agency-hub's API to record usage:
- `POST /api/claude/usage` on agency-hub
- Auth: management token
- Body: `{ "client_id": tenantID, "tokens_input": N, "tokens_output": N }`
- agency-hub writes to its `claude_usage` table

This keeps all agency DB writes in one service and avoids the portal needing a second DB connection.

**Option B:** Give portal a read connection to agency DB for usage queries only.

Use Option A for v1.

**GetClientBudget:** Call `GET /api/claude/budget/{client_id}` on agency-hub (management token auth). Returns `{ "monthly_budget": 150000, "tokens_used": 12400 }`.

**Budget check before every Claude call:**
```
remaining = monthly_budget - tokens_used
if remaining <= 0: return budget_exceeded error
```

### Prompt Builder (`internal/claude/prompt.go`)

Fetches current section content from client's Supabase using service role key (loaded via `ResolveTenant`).

```go
type PromptBuilder struct {
    tenantService *tenant.Service
}

func (pb *PromptBuilder) Build(ctx context.Context, tenantID, pageSlug, section string) (systemPrompt string, currentContent map[string]any, err error) {
    // 1. Get tenant config (Supabase URL + service role key)
    cfg, err := pb.tenantService.GetConfig(ctx, tenantID)

    // 2. Query client Supabase: SELECT sections FROM pages WHERE slug = $1
    //    Use service role key for this query
    //    Extract the specific section from the JSONB

    // 3. Build system prompt:
    systemPrompt = fmt.Sprintf(`You are a content editor for %s's website.
You are editing the "%s" section on the "%s" page.

Current section content (JSON):
%s

Respond ONLY with a valid JSON array of field changes.
Each object must have: "field", "current", "proposed", "notes".
Rules:
- Only change fields relevant to the instruction
- Do not add or remove fields from the JSON structure
- Return [] if the instruction is unclear, unsafe, or unrelated to content
- Do not include any text outside the JSON array`, clientName, section, pageSlug, serializedContent)
    return systemPrompt, currentContent, nil
}
```

### Claude Service (`internal/claude/service.go`)

```go
func (s *Service) Generate(ctx context.Context, tenantID string, req GenerateRequest) (*GenerateResponse, error) {
    // 1. Rate limit check
    if err := s.rl.Check(ctx, tenantID); err != nil {
        return nil, err   // caller maps to 429
    }

    // 2. Budget check
    budget, err := s.repo.GetClientBudget(ctx, tenantID)
    if err != nil { return nil, fmt.Errorf("budget_check_failed: %w", err) }
    if budget.TokensUsed >= budget.MonthlyBudget {
        return nil, fmt.Errorf("budget_exceeded")
    }

    // 3. Build prompt (fetches current content from client Supabase)
    systemPrompt, _, err := s.prompter.Build(ctx, tenantID, req.PageSlug, req.Section)
    if err != nil { return nil, fmt.Errorf("prompt_build_failed: %w", err) }

    // 4. Call Claude
    msg, err := s.anthropic.Messages.New(ctx, anthropic.MessageNewParams{
        Model:     anthropic.F(s.model),
        MaxTokens: anthropic.Int(1024),
        System: anthropic.F([]anthropic.TextBlockParam{
            {Type: anthropic.F(anthropic.TextBlockParamTypeText), Text: anthropic.F(systemPrompt)},
        }),
        Messages: anthropic.F([]anthropic.MessageParam{
            anthropic.UserMessageParam(anthropic.ContentBlockParamOfRequestTextBlock(req.Instruction)),
        }),
    })
    if err != nil {
        slog.Error("anthropic call failed", "tenant_id", tenantID, "error", err)
        return nil, fmt.Errorf("assistant_unavailable")
    }

    // 5. Parse response as []FieldChange
    var changes []FieldChange
    rawText := msg.Content[0].Text
    if err := json.Unmarshal([]byte(rawText), &changes); err != nil {
        slog.Error("failed to parse Claude JSON response", "raw", rawText)
        return nil, fmt.Errorf("assistant_parse_error")
    }

    // 6. Record usage async (fire and forget — never fail the request for this)
    tokensIn  := int(msg.Usage.InputTokens)
    tokensOut := int(msg.Usage.OutputTokens)
    go func() {
        if err := s.repo.RecordUsage(context.Background(), tenantID, tokensIn, tokensOut); err != nil {
            slog.Error("failed to record Claude usage", "tenant_id", tenantID, "error", err)
        }
    }()

    return &GenerateResponse{
        Changes:    changes,
        ModelUsed:  s.model,
        TokensUsed: tokensIn + tokensOut,
    }, nil
}
```

### Handler (`internal/claude/handler.go`)

```go
func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
    tenantID := r.Context().Value("tenant_id").(string)
    var req GenerateRequest
    // decode + validate...

    result, err := h.service.Generate(r.Context(), tenantID, req)
    if err != nil {
        switch {
        case strings.HasPrefix(err.Error(), "ratelimit_minute"):
            utils.Error(w, 429, "You're making requests too quickly. Please wait a moment.")
        case strings.HasPrefix(err.Error(), "ratelimit_hour"):
            utils.Error(w, 429, "Hourly limit reached. The assistant will be available again soon.")
        case err.Error() == "budget_exceeded":
            utils.Error(w, 429, "Your monthly content assistant limit has been reached. Your website team will be in touch.")
        case strings.HasPrefix(err.Error(), "prompt_build_failed"):
            utils.Error(w, 400, "Could not load the current page content. Try refreshing.")
        default:
            slog.Error("claude generate error", "error", err)
            utils.Error(w, 500, "The assistant is temporarily unavailable. Please try again.")
        }
        return
    }
    utils.JSON(w, 200, result)
}
```

### ISR Revalidation (`internal/revalidate/service.go`)

```go
func TriggerISR(ctx context.Context, clientSiteURL string, paths []string, secret string, clientID string) {
    body, _ := json.Marshal(map[string]any{"paths": paths})
    req, _ := http.NewRequestWithContext(ctx, "POST", clientSiteURL+"/api/revalidate", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Revalidate-Secret", secret)
    req.Header.Set("X-Client-ID", clientID)

    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        slog.Error("ISR revalidation failed", "site_url", clientSiteURL, "error", err)
        return  // non-blocking — content will still update on next hourly fallback
    }
    if resp.StatusCode != 200 {
        slog.Warn("ISR revalidation returned non-200", "status", resp.StatusCode)
    }
}
```

Called after every content update from the Claude assistant's Apply action. Also called by content save endpoints when the portal adds them.

**Client site URL source:** Stored in portal's tenant record. Add `site_url TEXT` column to tenants table. Populated when the tenant registers.

### Acceptance Criteria — Stage 4
- [ ] 5 requests in under 1 min → 6th returns 429 with "wait a moment" message
- [ ] 20 requests in under 1 hour → 21st returns 429 with "hourly limit" message
- [ ] Exhaust monthly budget (set low in test) → 429 with "monthly limit" message
- [ ] Claude returns valid JSON array → FieldChange objects parsed correctly
- [ ] Claude returns invalid JSON → 500 "temporarily unavailable" (not a raw parse error)
- [ ] Usage recorded in agency-hub after each successful call
- [ ] ISR triggered after each apply — client site revalidates

---

## Stage 5 — Frontend Foundation

Identical setup to agency-hub frontend plus:
```bash
pnpm add @supabase/supabase-js @supabase/ssr
pnpm add @tiptap/react @tiptap/pm @tiptap/starter-kit @tiptap/extension-link @tiptap/extension-image @tiptap/extension-placeholder
pnpm add @tiptap/extension-heading @tiptap/extension-blockquote @tiptap/extension-bullet-list @tiptap/extension-ordered-list
```

### Supabase Context (`src/contexts/supabase-context.tsx`)

```typescript
const SupabaseContext = createContext<SupabaseClient | null>(null)

export function SupabaseProvider({ children }: { children: ReactNode }) {
    const { user } = useAuth()  // user contains supabase_url + supabase_anon_key from JWT
    const [client, setClient] = useState<SupabaseClient | null>(null)

    useEffect(() => {
        if (user?.supabase_url && user?.supabase_anon_key) {
            setClient(createClient(user.supabase_url, user.supabase_anon_key))
        }
    }, [user?.supabase_url, user?.supabase_anon_key])

    return <SupabaseContext.Provider value={client}>{children}</SupabaseContext.Provider>
}

export function useTenantSupabase() {
    const client = useContext(SupabaseContext)
    if (!client) throw new Error("useTenantSupabase must be used within SupabaseProvider and after auth")
    return client
}
```

### Auth Context (portal-specific)

```typescript
interface PortalUser {
    user_id: string
    tenant_id: string
    email: string
    supabase_url: string
    supabase_anon_key: string
}
```

`GET /api/auth/profile` returns `PortalUser`. Everything else same as agency-hub.

### Routes

```
/connect          → ConnectPage (GuestRoute)
/auth/callback    → AuthCallbackPage (handles magic link exchange, GuestRoute)
/dashboard        → DashboardPage (ProtectedRoute)
/pages            → PagesListPage
/pages/:slug      → PageEditorPage
/blog             → BlogListPage
/blog/new         → NewPostPage
/blog/:id/edit    → EditPostPage
/media            → MediaPage
/forms            → FormsPage
/settings         → SettingsPage
/assistant        → AssistantPage
```

### Acceptance Criteria — Stage 5
- [ ] After magic link → exchange → SupabaseContext initialized with correct client
- [ ] `useTenantSupabase()` throws if called before auth (not null silently)
- [ ] /connect accessible without auth, all /dashboard+ routes require auth

---

## Stage 6 — Onboarding UI

### Connect Page (`/connect`)

Two-step UI:

**Step 1 — Form:**
- Title: "Access Your Dashboard"
- Subtitle: "Enter the access code from your website team"
- Fields: Access code (full-width text input), Email (email input)
- Submit button: "Continue →"
- Loading state: disabled button + spinner
- Error display: inline below form (NOT toast) with specific messages per error type

**Step 2 — Check Email screen:**
- Icon: email envelope
- Title: "Check your inbox"
- Body: "We sent a confirmation link to [email]. Click it to access your dashboard."
- Subtext: "The link expires in 24 hours."
- "Wrong email? Go back" link → returns to Step 1

**Error message mapping (from API response):**
```typescript
const errorMap: Record<string, string> = {
    "expired": "Your access code has expired. Ask your website team for a new one.",
    "used": "This access code has already been used. Contact your website team.",
    "invalid": "Invalid access code. Double-check for typos and try again.",
    "not_setup": "This account isn't set up yet. Contact your website team.",
}
```

### Auth Callback Page (`/auth/callback`)

Handles the magic link redirect from Supabase:
1. Extract `access_token` from URL fragment (`#access_token=...`)
2. Call `POST /api/auth/exchange` with the token
3. On success: navigate to `/dashboard`
4. On error: navigate to `/connect` with error query param

### Acceptance Criteria — Stage 6
- [ ] Valid connect → check email screen shown
- [ ] Each specific error shows the correct user-friendly message
- [ ] Magic link callback → dashboard reached
- [ ] "Wrong email? Go back" → resets to step 1 with empty form

---

## Stage 7 — Portal Shell

### Layout

**PortalSidebar nav items:**
```
Dashboard    /dashboard
Pages        /pages
Blog         /blog
Media        /media
Forms        /forms
Settings     /settings
Assistant    /assistant
```

Footer: User email, "View Site →" link (opens client site in new tab), Logout button

**"View Site" URL:** From `user.supabase_url` — wait, this is the Supabase URL not site URL. Need the site URL from... the portal backend's tenant record.
- Add `GET /api/tenant/info` endpoint: returns `{ site_url: "...", client_name: "..." }`
- Cache in TanStack Query, show in sidebar header

### Shared Components

**`SaveIndicator`:**
- States: `idle | saving | saved | error`
- `saving`: spinner + "Saving..."
- `saved`: checkmark + "Saved" (auto-clears after 2s)
- `error`: X icon + "Error saving. Try again."
- Used in all content editors

**`EmptyState({ icon, title, description, action })`:**
Standard empty state for tables/lists.

**`ConfirmDialog({ title, description, onConfirm, dangerous? })`:**
Wraps shadcn `AlertDialog`. If `dangerous=true`, confirm button is red.

---

## Stage 8 — Page & Section Editors

### Pages List (`/pages`)

Fetches from client's Supabase: `SELECT id, slug, title, is_published, updated_at FROM pages ORDER BY slug`

Table columns: Title, Slug, Status (Published/Draft badge), Last Updated, Edit button

**Empty state:** "No pages yet. Pages are created automatically when your site is set up."

### Page Editor (`/pages/:slug`)

Fetches: `SELECT * FROM pages WHERE slug = $1`

**Layout:** Page title at top. Below: list of section editors based on which keys exist in `sections` JSONB.

**Section editor dispatch logic:**
```typescript
const sectionEditors: Record<string, React.ComponentType<{data: any, onChange: (data: any) => void}>> = {
    hero: HeroEditor,
    features: FeaturesEditor,
    about: AboutEditor,
    testimonials: TestimonialsEditor,
    cta: CTAEditor,
}
```

Each section editor:
- Shows current values in editable fields
- Has a "Save section" button
- On save: `UPDATE pages SET sections = jsonb_set(sections, '{key}', $1), updated_at = NOW() WHERE slug = $2`
- After save: call portal backend `POST /api/revalidate` with path `/`
- Shows `SaveIndicator`

**`HeroEditor` fields:**
- Headline (text input)
- Subheadline (textarea)
- CTA Label (text input)
- CTA URL (text input, validates as URL)

**`FeaturesEditor` fields:**
- Headline (text input)
- Features list (repeatable): icon name (Lucide icon select), title, description
- Add item button, remove item button, reorder (drag or up/down arrows)

**`AboutEditor` fields:**
- Headline (text input)
- Body (textarea, multiline)
- Image (MediaPickerModal button showing current image or "No image")

**`TestimonialsEditor` fields:**
- Headline (text input)
- Testimonials list (repeatable): quote, author name, author role, avatar (optional, MediaPickerModal)

**`CTAEditor` fields:**
- Headline, subheadline, button label, button URL

**`SectionPublishToggle`:** Toggle `pages.is_published`. When toggled: `UPDATE pages SET is_published = $1 WHERE slug = $2`. Trigger ISR after toggle.

### Acceptance Criteria — Stage 8
- [ ] Edit hero headline → save → `pages.sections.hero.headline` updated in Supabase
- [ ] Save indicator cycles through saving → saved → idle correctly
- [ ] Features editor: add item → shows in list; remove item → removed from list
- [ ] ISR triggered after every section save
- [ ] Publish toggle updates `is_published` in Supabase

---

## Stage 9 — Blog Editor

### Posts List (`/blog`)

Fetches: `SELECT id, slug, title, is_published, author_name, published_at, created_at FROM posts ORDER BY created_at DESC`

Table: Title, Slug, Status, Author, Published Date, Edit + Delete actions

**Delete:** Confirmation dialog → `DELETE FROM posts WHERE id = $1` → remove from table optimistically

### Post Editor

**Post meta form fields:**
- Slug (text, auto-generated from title via `slugify(title)` but editable)
- Excerpt (textarea, max 300 chars, character counter shown)
- Cover image (MediaPickerModal)
- Author name (text input)
- SEO title (text input, optional override)
- SEO description (textarea, optional)

**Slug validation:** Must match `^[a-z0-9]+(?:-[a-z0-9]+)*$`. Check uniqueness on blur: `SELECT COUNT(*) FROM posts WHERE slug = $1 AND id != $2`.

**Tiptap Editor configuration:**
```typescript
const editor = useEditor({
    extensions: [
        StarterKit.configure({ heading: { levels: [2, 3] } }),
        Link.configure({ openOnClick: false, HTMLAttributes: { rel: 'noopener noreferrer' } }),
        Image,
        Placeholder.configure({ placeholder: 'Start writing...' }),
    ],
    content: post?.content ?? '',
    onUpdate: ({ editor }) => setContent(editor.getHTML()),
})
```

**Toolbar buttons:** Bold, Italic, H2, H3, Bullet List, Ordered List, Blockquote, Link, Image (opens MediaPickerModal), Undo, Redo

**Draft/Publish toggle:**
- Draft → sets `is_published = false`, `published_at = NULL`
- Publish → sets `is_published = true`, `published_at = NOW()` (if null)
- Once published, `published_at` is NOT reset if toggled back to draft

**Auto-save:** Debounce 2 seconds after typing stops → save content to Supabase. Show `SaveIndicator`. Do NOT save on every keystroke.

**New post flow:**
- Slug pre-populated from title (live update as user types title, but slug field is editable)
- Content starts empty
- Default status: Draft
- On first save: INSERT into posts
- Subsequent saves: UPDATE

### Acceptance Criteria — Stage 9
- [ ] Auto-save fires 2s after typing stops (debounced)
- [ ] Duplicate slug: inline error on blur
- [ ] Publish toggle: published_at set on first publish, NOT cleared on unpublish
- [ ] Cover image set via MediaPickerModal
- [ ] Tiptap toolbar all buttons functional
- [ ] Delete post with confirmation → removed from list

---

## Stage 10 — Media Library

### Media List

Fetches from Supabase Storage: `supabase.storage.from('media').list()` + join with `media` table metadata.

**Grid:** 3-4 columns, each cell:
- Thumbnail (Next.js img with object-cover)
- Filename (truncated)
- File size (formatted: "245 KB")
- Copy URL button (icon, copies public URL)
- Delete button (icon, confirmation required)

**Upload:**
- Drag-and-drop zone: accepts images (jpeg, png, webp, gif, svg) and max 5MB
- File picker button as fallback
- On drop/select:
  1. Validate: type must be image, size < 5MB — show inline error if invalid
  2. Upload to Supabase Storage: `supabase.storage.from('media').upload(filename, file)`
  3. Get public URL: `supabase.storage.from('media').getPublicUrl(filename)`
  4. Insert metadata into `media` table: `{ filename, url, mime_type, size_bytes }`
  5. Show upload progress indicator
  6. Add to grid optimistically

**Delete:**
1. Delete from Supabase Storage: `supabase.storage.from('media').remove([filename])`
2. Delete from `media` table
3. Remove from grid optimistically

**`MediaPickerModal`:**
- Reusable modal that renders the grid in selection mode
- On item click: calls `onSelect(url)` callback and closes
- Used by: `AboutEditor` (image field), `TestimonialsEditor` (avatar), `PostEditor` (cover image, inline image insert)

### Acceptance Criteria — Stage 10
- [ ] Upload: valid image → appears in grid with correct filename + size
- [ ] Upload: > 5MB file → inline error "File too large (max 5MB)"
- [ ] Upload: non-image file → inline error "Only image files are supported"
- [ ] Delete: confirmation → removed from grid and Storage
- [ ] MediaPickerModal: select image → URL returned to caller

---

## Stage 11 — Claude Assistant UI

### AssistantPanel

Floating panel accessible from any page editor via a "✨ Assistant" button in the page toolbar. Renders as a `Sheet` (slides in from right, 480px wide).

**States:**
```
idle       → Instruction form visible
loading    → Spinner, "Generating suggestions...", form disabled
preview    → DiffPreview + ApplyBar visible, instruction form hidden
applied    → Success message, ISR status, "Make another change" button
error      → Error message with specific text per error type, retry button
```

**InstructionForm:**
- Page selector (dropdown — lists available pages from Supabase)
- Section selector (dropdown — populated based on selected page's sections JSONB keys)
- Instruction textarea (min 5 chars, max 500 chars, character counter)
- "Generate" button
- "Powered by Claude" attribution line (small, muted)

**DiffPreview:**

Side-by-side diff table:

| Field | Current | Proposed |
|-------|---------|----------|
| headline | We help businesses | Summer Sale — 40% Off |
| subheadline | ... | ... |

Each row uses green background for "proposed" cell. If `proposed === current` (no change for that field), show a muted "—" in proposed column.

**ApplyBar:**
- "Apply Changes" button (primary, full-width)
- "Discard" button (text button, below)
- Token count: "~142 tokens used"
- On Apply:
  1. Write only the changed fields to Supabase (merge into existing JSONB, not replace)
  2. Call `POST /api/revalidate` via portal backend
  3. Show "Applied! Your site is updating..." for 2s, then switch to idle state
- On Discard: return to idle, clear state

**RateLimitBanner:**
- Renders instead of the instruction form when 429 received
- Per-error type text:
  - minute: "You're making requests too quickly. Wait a moment and try again."
  - hour: "You've reached your hourly limit. The assistant will be available again soon."
  - budget: "You've reached your monthly content assistant limit. Your website team has been notified."
- Show relevant icon + colored background (yellow for rate limit, orange for budget)

### Acceptance Criteria — Stage 11
- [ ] Instruction → Generate → DiffPreview shows side-by-side changes
- [ ] Apply → Supabase updated → ISR triggered → success state shown
- [ ] Discard → returns to instruction form (Supabase NOT modified)
- [ ] Rate limit 429 → RateLimitBanner shown with correct message
- [ ] Budget 429 → budget-specific banner shown
- [ ] Empty changes array from Claude → show "No changes suggested. Try a more specific instruction."
- [ ] Form disabled during loading

---

## Stage 12 — Secondary Features

### Form Submissions (`/forms`)

Table: Date, Form name, Data preview (first 60 chars of JSON), Read/Unread badge, View button

**Detail view (Sheet):** Full JSON data rendered as a key-value list. Mark as read on open: `UPDATE form_submissions SET is_read = true WHERE id = $1`.

**Unread count badge** on sidebar nav item "Forms".

### Settings (`/settings`)

Tabbed: General | SEO | Social | Navigation

**General tab:**
- Site name, tagline
- Logo upload (MediaPickerModal → sets `site_settings.logo_url`)
- Contact email, phone, address
- Each field: saves individually on blur OR a "Save General" button at bottom

**SEO tab:**
- SEO title, SEO description, OG image (MediaPickerModal)
- Character counters: SEO title (60 chars max), SEO description (160 chars max)

**Social tab:**
- Twitter, Instagram, Facebook, LinkedIn URLs (validated as URLs on save)

**Navigation tab:**
- List of nav items with label + URL
- Reorder by drag-and-drop
- Add new item button
- Remove item button
- External link toggle (opens in new tab)
- "Save Navigation" button → bulk UPDATE nav_items

All settings write to `site_settings` table via `supabase.from('site_settings').upsert()`.

### Dashboard (`/dashboard`)

Quick stats row:
- "Last edited" section (from most recent `pages.updated_at` or `posts.updated_at`)
- "Unread form submissions" count
- "Published posts" count
- "Draft posts" count

Quick action cards:
- "Edit Home Page" → `/pages/home`
- "New Blog Post" → `/blog/new`
- "Upload Media" → `/media`
- "View Forms" → `/forms`

### Acceptance Criteria — Stage 12
- [ ] Form submission: mark as read → badge disappears
- [ ] Settings: change site name → save → reflected in site_settings table
- [ ] Nav editor: reorder + save → nav_items table updated with new order values
- [ ] OG image set via MediaPickerModal
- [ ] Dashboard quick actions navigate to correct pages

---

## Stage 13 — QA & Deploy

### Pre-Deploy Checklist
- [ ] Management token validation on startup confirmed in Railway logs
- [ ] Onboarding flow end-to-end: token → email → confirm → dashboard
- [ ] Service role key never appears in any API response body or header
- [ ] JWT does NOT contain service role key (only anon key)
- [ ] Claude rate limits work correctly (test all three: minute, hour, budget)
- [ ] ISR revalidation actually triggers on client-site (check Next.js logs)
- [ ] CSRF enforced on all mutations
- [ ] Tenant isolation: simulate two tenants, verify tenant A cannot read tenant B's data
- [ ] All 429 responses have human-readable, non-technical messages
- [ ] Frontend: no TanStack Query key collisions between features
- [ ] Frontend: all loading/error/empty states implemented
- [ ] `useTenantSupabase()` throws clearly if called before auth (not a silent null)
