# CLAUDE.md — agency-hub
> Internal agency management tool. Dagim + Matt + team only.
> RBAC: super_admin · admin · moderator

---

## What This App Is

agency-hub is the agency's internal command center. It manages client records, generates the tokens that bind client deployments to the agency system, tracks projects, and monitors deploy health. It has no public-facing component. The only people with access are agency team members.

---

## Stack

**Backend (`api/`)**
- Go 1.24+ · Chi v5
- PostgreSQL via Supabase (agency's own project)
- JWT: 15-min access token, 7-day refresh, HTTP-only cookies
- Upstash Redis: rate limiting (sliding window) + session tracking
- Resend: transactional email (invites, role notifications)
- Swagger (swag): API docs at `/swagger/index.html`
- slog: structured logging throughout
- go-playground/validator v10: request body validation
- Feature-based folder: `internal/<feature>/{model,repository,service,handler,routes}.go`

**Frontend (`web/`)**
- React 19 · TypeScript (strict) · Vite
- Tailwind CSS v4 · shadcn/ui · Radix UI
- TanStack Query v5 · React Router v7
- React Hook Form + Zod · Framer Motion · Sonner · Lucide React
- Feature-based folder: `src/features/<feature>/{pages,components,hooks,services,types}/`

---

## Design → Implementation Workflow (Variants)

**This is the canonical design process for this project. Follow it for every new page and component type.**

### Rule 1 — Variants is the visual source of truth
When a Variants export is pasted into this session, Claude's job is to **adapt the code to project conventions** — not redesign it. Preserve:
- Layout and spacing decisions exactly
- Color choices (map to Tailwind v4 utilities or CSS variables)
- Typography scale
- Component composition (which elements are grouped together)

Do NOT:
- Suggest alternative layouts
- Change the visual hierarchy
- Replace design decisions with "standard" shadcn defaults if the Variants design differs

### Rule 2 — First design locks the pattern
The first Variants-derived version of any component type becomes the canonical pattern for this project. Examples:
- First card design → all cards follow that pattern
- First table design → all tables follow that pattern
- First modal design → all modals follow that pattern
- First sidebar nav → navigation stays consistent

When building a new feature, check existing adapted components first. Reuse before creating.

### Rule 3 — Paste → Adapt → Commit
```
1. Dagim pastes Variants export into the session
2. Claude adapts it:
   a. Replace non-shadcn UI elements with shadcn equivalents
   b. Type all props with TypeScript interfaces (no 'any')
   c. Move to correct feature folder: src/features/<feature>/components/
   d. Replace any hardcoded values with props or data-driven values
   e. Ensure accessibility: aria labels, keyboard nav, focus rings
   f. Export as named export (not default)
3. Dagim reviews, confirms, commits → this is now the pattern
```

### Variants Paste Prompt (use this every time)
```
Here is a Variants export for [component name].
Adapt it to agency-hub conventions:

Stack: React 19 + TypeScript (strict) + Tailwind v4 + shadcn/ui
Target file: src/features/[feature]/components/[ComponentName].tsx

Rules:
- Preserve the layout, spacing, and visual decisions exactly
- Replace non-shadcn elements with shadcn equivalents where they exist
- Type all props (no 'any')
- Use Lucide React for icons
- Named export only
- Add Framer Motion entrance animation if the component is a card or list item

[paste Variants code here]
```

---

## RBAC — Roles & Access

```
super_admin   Full access. Can manage roles, view all data.
admin         Full operational access. Create clients, generate tokens, manage projects.
moderator     Read-only across all areas. No write operations.
```

**JWT claims this app issues:**
```go
type Claims struct {
    UserID   string `json:"user_id"`
    Email    string `json:"email"`
    Role     string `json:"role"`     // "super_admin" | "admin" | "moderator"
    IsActive bool   `json:"is_active"`
    jwt.RegisteredClaims
}
```

**RBAC middleware pattern:**
```go
// Use in routes.go:
r.With(middleware.RequireRole("admin", "super_admin")).Post("/clients", h.Create)
r.With(middleware.RequireRole("super_admin")).Post("/team/roles/assign", h.AssignRole)
```

---

## Token Architecture

Two types of tokens are managed by this app:

**`management_token`** (system-level):
- Generated when admin creates a client record
- 32-byte hex string, stored SHA-256 hashed in DB
- Shown ONCE in plaintext — goes into client-portal + client-site env vars
- Never expires unless explicitly rotated
- Used by portal/site backends to authenticate with agency-hub's API

**`connection_token`** (user-level):
- Generated separately when admin is ready to onboard the client user
- 7-day expiry, one-time use, consumed on first portal login
- Sent to the actual client person via email
- Portal backend calls `POST /api/validate-connection-token` using management_token as auth

Both tokens are generated using `crypto/rand` (32 bytes → hex), stored hashed. **Never store plaintext tokens in the database.**

---

## Security Rules — Non-Negotiable

- CSRF token required on ALL POST/PUT/PATCH/DELETE routes
- Input validation via `validator.New().Struct()` on every request body
- Rate limiting on all auth routes: 5 req/min (Upstash Redis, sliding window)
- Rate limiting on all authenticated routes: 30 req/min
- Account lockout: 5 failed logins → 15-min lockout (Redis-backed)
- `DB_SSLMODE=require` enforced in production config
- Security headers middleware on all routes: X-Content-Type-Options, X-Frame-Options, CSP, HSTS
- Audit log every: login, logout, role change, token generation, token revocation, client create/update
- Errors: log internally with slog, return generic message to client — never leak internals

---

## Backend Conventions

### Adding a New Feature (Go)
```
internal/<feature>/
├── model.go        → DB row structs, request types, response types, ToResponse() methods
├── repository.go   → All DB queries (parameterized only), NewRepository(db *sql.DB)
├── service.go      → Business logic, calls repository + mailer + audit
├── handler.go      → HTTP handlers with Swagger annotations, calls service
└── routes.go       → Chi route registration, middleware wiring
```

**Prompt to scaffold a new feature:**
```
Scaffold the [feature] feature for agency-hub's Go backend.

Context:
- Go 1.24+ · Chi v5 · PostgreSQL (database/sql) · slog logging
- validator v10 for input validation · Swagger annotations required
- Feature folder: internal/[feature]/

Create: model.go, repository.go, service.go, handler.go, routes.go

Feature description:
[describe what it does]

DB schema (relevant tables):
[paste migration SQL]

Endpoints needed:
[list method + path + who can call it + what it does]

Reference our existing pattern — here is internal/client/handler.go:
[paste when available]
```

### Error Response Format
```go
// Always return JSON, never plain text:
type ErrorResponse struct {
    Error string `json:"error"`
}
// Internal details go to slog, never to the client response
```

### Swagger Annotation Format
```go
// Create godoc
// @Summary     Create a new client
// @Tags        clients
// @Accept      json
// @Produce     json
// @Param       body body CreateClientRequest true "Client data"
// @Success     201 {object} ClientResponse
// @Failure     400 {object} ErrorResponse
// @Failure     401 {object} ErrorResponse
// @Failure     403 {object} ErrorResponse
// @Router      /clients [post]
```

---

## Frontend Conventions

### Adding a New Page
```
src/features/<feature>/
├── pages/<PageName>.tsx       → route-level component, uses hooks, no direct API calls
├── components/<Name>.tsx      → feature-specific UI components
├── hooks/use-<feature>.ts     → TanStack Query wrappers
├── services/<feature>.service.ts → axios calls
└── types/index.ts             → TypeScript interfaces for this feature
```

**Prompt to scaffold a new page:**
```
Add a new page to agency-hub: [Page Name]

Route: [path]
Feature folder: src/features/[feature]/

Page purpose: [describe]

API endpoints it uses (from Swagger):
[list]

Requirements:
- Auth guard: [ProtectedRoute | RoleProtectedRoute role="admin"]
- Loading state: shadcn Skeleton matching the layout
- Error state: Sonner toast
- Empty state: [describe]
- Responsive layout
- Match existing component patterns (reference [similar existing component])

[paste Variants export if available]
```

### Key Hooks Pattern
```typescript
// Always wrap TanStack Query — never call services directly from components
export const CLIENTS_KEY = ['clients'] as const

export function useClients() {
  return useQuery({ queryKey: CLIENTS_KEY, queryFn: clientsService.getAll })
}

export function useCreateClient() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: clientsService.create,
    onSuccess: () => { qc.invalidateQueries({ queryKey: CLIENTS_KEY }); toast.success('Client created') },
    onError:   () => { toast.error('Failed to create client') },
  })
}
```

### Auth Context
Single source of truth for auth state. Access via `useAuth()` hook only. Never store auth state locally in components.

### API Client (`src/lib/axios.ts`)
All API calls go through this client. It handles:
- CSRF token attachment (fetched once on app load, attached to all mutation requests)
- 401 → attempt token refresh → retry → redirect to login if refresh fails
- 429 → toast "Too many requests"
- 500 → silent log (no toast — don't alarm internal users)

---

## Do Not

**Backend:**
- Do not use string interpolation in SQL queries — parameterized only
- Do not return stack traces or Go error messages to HTTP clients
- Do not skip Swagger annotations on any handler
- Do not skip audit logging on any action that modifies data
- Do not store plaintext tokens — always SHA-256 hash before writing to DB

**Frontend:**
- Do not use `useEffect` for data fetching — always TanStack Query
- Do not create new UI primitives — use shadcn/ui
- Do not use `any` — always type properly
- Do not import across feature folders — use `src/types/` for shared types
- Do not deviate from the Variants-established visual patterns
- Do not put API keys in any frontend file

---

## DB Schema Reference (Agency Supabase)

```sql
users(id, email, name, role, is_active, created_at)
role_requests(id, requested_by, target_email, requested_role, status, reviewed_by, created_at)
clients(id, name, business_name, email, phone, plan_tier, status, notes,
        management_token_hash, management_token_prefix,
        connection_token_hash, connection_token_expires_at, connection_token_used_at,
        client_supabase_project_ref, client_supabase_url,
        claude_monthly_token_budget, claude_model,
        portal_url, site_url, site_domain, domain_registrar, dns_provider,
        railway_service_url, vercel_project_id,
        created_by, created_at)
projects(id, client_id, name, description, status, linear_project_url,
         notion_page_url, estimated_delivery, actual_delivery, assigned_to, created_at)
deploy_records(id, client_id, frontend_url, backend_url, supabase_ref,
               last_checked_at, health_status, created_at)
audit_logs(id, actor_id, action, entity_type, entity_id, metadata, ip, created_at)
claude_usage(id, client_id, year_month, requests, tokens_input, tokens_output,
             last_used_at, created_at)
```

---

## Environment Variables Reference

```env
# Database
SUPABASE_DB_URL=postgresql://postgres:[password]@[host]:5432/postgres
DB_SSLMODE=require

# Auth
JWT_SECRET=
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# Redis
UPSTASH_REDIS_URL=
UPSTASH_REDIS_TOKEN=

# Email
RESEND_API_KEY=
RESEND_FROM=noreply@youragency.com

# App
ENVIRONMENT=development
FRONTEND_URL=http://localhost:5173
PORT=8080
```
