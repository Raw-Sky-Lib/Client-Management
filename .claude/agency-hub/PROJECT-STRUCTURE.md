# agency-hub — Project Structure
> Every file and folder that will exist in this repo. Plan before you build.

---

## Repository Root

```
agency-hub/
├── api/                    # Go backend
├── web/                    # React frontend
├── .github/
│   └── workflows/
│       └── deploy.yml      # Auto-deploy main → Railway (backend) + Vercel (frontend)
└── README.md
```

---

## Backend (`api/`)

```
api/
├── cmd/
│   ├── app/
│   │   └── main.go                   # Entry point: load config, connect DB/Redis,
│   │                                 #   validate startup, wire routes, serve
│   ├── admin/
│   │   └── main.go                   # CLI: bootstrap first super_admin
│   │                                 #   Usage: go run cmd/admin/main.go promote
│   │                                 #          --email admin@agency.com --role super_admin
│   └── migrate/
│       └── main.go                   # Run SQL migrations from supabase/migrations/
│
├── internal/
│   │
│   ├── auth/
│   │   ├── model.go                  # LoginRequest, LoginResponse, Claims,
│   │   │                             #   RefreshTokenRecord, CSRFToken
│   │   ├── repository.go             # StoreRefreshToken, GetRefreshToken,
│   │   │                             #   DeleteRefreshToken, IncrementLoginAttempts,
│   │   │                             #   GetLoginAttempts, ResetLoginAttempts
│   │   ├── service.go                # Login, Logout, RefreshTokens, GenerateCSRF,
│   │   │                             #   ValidateCSRF, IssueTokenPair
│   │   ├── handler.go                # POST /auth/login, POST /auth/logout,
│   │   │                             #   POST /auth/refresh, GET /auth/csrf,
│   │   │                             #   GET /auth/profile
│   │   └── routes.go
│   │
│   ├── team/
│   │   ├── model.go                  # User, CreateUserRequest, UpdateUserRequest,
│   │   │                             #   UserResponse, RoleRequest, InviteRequest
│   │   ├── repository.go             # GetUserByID, GetUserByEmail, GetAllUsers,
│   │   │                             #   CreateUser, UpdateUserRole, DeactivateUser,
│   │   │                             #   CreateRoleRequest, GetRoleRequest,
│   │   │                             #   UpdateRoleRequestStatus
│   │   ├── service.go                # InviteTeamMember, AssignRole, RequestRolePromotion,
│   │   │                             #   ApproveRoleRequest, RejectRoleRequest
│   │   ├── handler.go                # GET /team, POST /team/invite,
│   │   │                             #   POST /team/roles/assign (super_admin only),
│   │   │                             #   POST /team/roles/requests (admin+),
│   │   │                             #   POST /team/roles/requests/:id/approve,
│   │   │                             #   POST /team/roles/requests/:id/reject
│   │   └── routes.go
│   │
│   ├── client/
│   │   ├── model.go                  # Client, CreateClientRequest, UpdateClientRequest,
│   │   │                             #   ClientResponse, TokenGenerationResponse,
│   │   │                             #   ConnectionTokenResponse, ValidateTokenRequest
│   │   ├── repository.go             # CreateClient, GetClientByID, GetAllClients,
│   │   │                             #   UpdateClient, UpdateManagementToken,
│   │   │                             #   UpdateConnectionToken, MarkConnectionTokenUsed,
│   │   │                             #   GetClientByManagementTokenHash,
│   │   │                             #   ValidateConnectionToken
│   │   ├── service.go                # CRUD operations, GenerateManagementToken,
│   │   │                             #   RotateManagementToken, GenerateConnectionToken,
│   │   │                             #   RevokeConnectionToken, ValidateManagementToken
│   │   ├── handler.go                # GET /clients, POST /clients,
│   │   │                             #   GET /clients/:id, PUT /clients/:id,
│   │   │                             #   POST /clients/:id/management-token (generate),
│   │   │                             #   POST /clients/:id/management-token/rotate,
│   │   │                             #   POST /clients/:id/connection-token,
│   │   │                             #   DELETE /clients/:id/connection-token (revoke),
│   │   │                             #   GET /validate-management-token (used by portal/site)
│   │   └── routes.go
│   │
│   ├── project/
│   │   ├── model.go                  # Project, CreateProjectRequest, UpdateProjectRequest,
│   │   │                             #   ProjectResponse, ProjectStatus enum
│   │   ├── repository.go             # CRUD, GetByClientID, UpdateStatus
│   │   ├── service.go                # CRUD with audit logging
│   │   ├── handler.go                # GET /projects, POST /projects,
│   │   │                             #   GET /projects/:id, PUT /projects/:id,
│   │   │                             #   PATCH /projects/:id/status,
│   │   │                             #   GET /clients/:id/projects
│   │   └── routes.go
│   │
│   ├── deploy/
│   │   ├── model.go                  # DeployRecord, HealthStatus, PingResult
│   │   ├── repository.go             # UpsertDeployRecord, GetByClientID,
│   │   │                             #   UpdateHealthStatus
│   │   ├── service.go                # PingClientSite (HTTP GET to client /health),
│   │   │                             #   UpdateDeployInfo
│   │   ├── handler.go                # GET /clients/:id/deploy,
│   │   │                             #   PUT /clients/:id/deploy,
│   │   │                             #   POST /clients/:id/deploy/ping
│   │   └── routes.go
│   │
│   ├── dashboard/
│   │   ├── model.go                  # DashboardStats: active clients count,
│   │   │                             #   projects by status, recent clients,
│   │   │                             #   claude usage summary
│   │   ├── repository.go             # Aggregate queries across all tables
│   │   ├── service.go
│   │   ├── handler.go                # GET /dashboard
│   │   └── routes.go
│   │
│   ├── audit/
│   │   ├── model.go                  # AuditLog, AuditAction constants
│   │   ├── repository.go             # InsertLog, GetLogs (paginated, filterable)
│   │   └── service.go                # LogAction(ctx, actorID, action, entityType,
│   │                                 #   entityID, metadata) — called by other services
│   │                                 # No handler — audit is internal only
│   │
│   ├── mailer/
│   │   ├── model.go                  # Email template types
│   │   └── service.go                # SendInvite, SendRoleApproved, SendRoleRejected,
│   │                                 #   SendBudgetAlert (Claude usage)
│   │                                 # Uses Resend HTTP API
│   │
│   ├── config/
│   │   └── config.go                 # Load all env vars, validate required ones,
│   │                                 #   return typed Config struct
│   │
│   ├── database/
│   │   ├── db.go                     # Connect (sql.Open + ping), close
│   │   └── migrate.go                # Read migration files, execute in order
│   │
│   ├── middleware/
│   │   ├── auth.go                   # ExtractJWT → inject Claims into context
│   │   ├── rbac.go                   # RequireRole(roles ...string) middleware factory
│   │   ├── csrf.go                   # ValidateCSRF checks X-CSRF-Token header
│   │   ├── ratelimit.go              # NewRateLimiter(redis, limit, window) factory
│   │   │                             #   uses Upstash Redis sliding window
│   │   ├── security.go               # Sets: X-Content-Type-Options, X-Frame-Options,
│   │   │                             #   Strict-Transport-Security, CSP, Referrer-Policy
│   │   └── logger.go                 # Log every request: method, path, status, latency
│   │
│   └── utils/
│       ├── crypto.go                 # GenerateToken() → 32-byte hex,
│       │                             #   HashToken(token) → SHA-256 hex,
│       │                             #   TokenPrefix(token) → first 8 chars
│       ├── response.go               # JSON(w, status, data), Error(w, status, msg)
│       └── errors.go                 # AppError type, HTTP status mapping
│
├── pkg/
│   └── logger/
│       └── logger.go                 # Initialise slog with JSON handler (prod)
│                                     #   or text handler (dev)
│
├── docs/                             # Auto-generated by: swag init -g cmd/app/main.go
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
│
├── supabase/
│   └── migrations/
│       ├── 001_create_users.sql
│       ├── 002_create_role_requests.sql
│       ├── 003_create_clients.sql
│       ├── 004_create_projects.sql
│       ├── 005_create_deploy_records.sql
│       ├── 006_create_audit_logs.sql
│       └── 007_create_claude_usage.sql
│
├── .env.example
├── .air.toml                         # Hot reload config for local dev
├── railway.toml                      # Railway deployment config
└── go.mod
```

### Key File Notes

**`cmd/app/main.go` startup sequence:**
```
1. Load config (panic if required env vars missing)
2. Connect DB (panic if unreachable)
3. Connect Redis (panic if unreachable)
4. Run pending migrations
5. Wire all feature handlers + middleware
6. Start HTTP server
```

**`internal/client/handler.go` — `GET /validate-management-token`:**
This endpoint is called by client-portal and client-site on startup to verify their management token is still valid. It must:
- Accept `Authorization: Bearer <management_token>` + `X-Client-ID: <client_id>` header
- Hash the token, look up client by ID, compare hashes
- Return `{ client_id, client_name, status }` or 401
- This endpoint does NOT require CSRF (it's an API-to-API call from the portal/site backend)
- Rate limit: 10 req/min per IP (prevent token enumeration)

---

## Frontend (`web/`)

```
web/
├── public/
│   ├── favicon.ico
│   └── logo.svg                      # Agency logo
│
├── src/
│   │
│   ├── components/
│   │   ├── layout/
│   │   │   ├── AppLayout.tsx         # Root layout: sidebar left, content right
│   │   │   │                         #   Wraps all authenticated pages
│   │   │   ├── AppSidebar.tsx        # Left nav — role-aware link visibility
│   │   │   │                         #   Sections: Dashboard, Clients, Projects, Team
│   │   │   │                         #   Footer: user avatar, role badge, logout
│   │   │   └── AppHeader.tsx         # Top bar: page title, breadcrumb, user menu
│   │   │
│   │   ├── guards/
│   │   │   ├── ProtectedRoute.tsx    # Redirect to /login if no valid session
│   │   │   ├── GuestRoute.tsx        # Redirect to /dashboard if already authed
│   │   │   └── RoleProtectedRoute.tsx # Redirect to /dashboard if wrong role
│   │   │
│   │   └── ui/                       # shadcn/ui components (run: npx shadcn@latest add ...)
│   │       # Required components:
│   │       # button, card, input, label, badge, table, dialog, sheet,
│   │       # dropdown-menu, avatar, skeleton, toast (sonner), tooltip,
│   │       # select, textarea, form, separator, alert, progress,
│   │       # tabs, popover, command (for search)
│   │
│   ├── config/
│   │   └── routes.tsx                # Central route registry with lazy imports
│   │                                 # All routes defined here, nowhere else
│   │
│   ├── contexts/
│   │   └── auth-context.tsx          # AuthContext: user, isLoading, login, logout
│   │                                 # Backed by TanStack Query (GET /auth/profile)
│   │                                 # Single source of truth — no local auth state
│   │
│   ├── features/
│   │   │
│   │   ├── auth/
│   │   │   ├── pages/
│   │   │   │   └── LoginPage.tsx     # Full-page login — not inside AppLayout
│   │   │   ├── components/
│   │   │   │   └── LoginForm.tsx     # Email + password form, error states,
│   │   │   │                         #   account lockout message
│   │   │   ├── hooks/
│   │   │   │   └── use-auth.ts       # useLogin(), useLogout() mutations
│   │   │   ├── services/
│   │   │   │   └── auth.service.ts   # login(), logout(), getProfile(), getCSRF()
│   │   │   └── types/
│   │   │       └── index.ts          # LoginRequest, User, AuthState
│   │   │
│   │   ├── dashboard/
│   │   │   ├── pages/
│   │   │   │   └── DashboardPage.tsx # Composes all dashboard widgets
│   │   │   ├── components/
│   │   │   │   ├── StatsGrid.tsx     # 4-stat overview row (active clients, live sites,
│   │   │   │   │                     #   open projects, this-month Claude usage)
│   │   │   │   ├── StatCard.tsx      # Individual stat: label, value, trend, icon
│   │   │   │   ├── RecentClients.tsx # Last 5 clients with status badges + quick links
│   │   │   │   └── ProjectsOverview.tsx # Projects grouped by status (mini kanban)
│   │   │   ├── hooks/
│   │   │   │   └── use-dashboard.ts  # useDashboardStats()
│   │   │   └── services/
│   │   │       └── dashboard.service.ts
│   │   │
│   │   ├── clients/
│   │   │   ├── pages/
│   │   │   │   ├── ClientsPage.tsx       # Searchable, filterable client list
│   │   │   │   ├── ClientDetailPage.tsx  # Client profile + all sub-sections
│   │   │   │   └── NewClientPage.tsx     # Multi-step new client form
│   │   │   ├── components/
│   │   │   │   ├── ClientsTable.tsx      # Sortable table with status, plan, actions
│   │   │   │   ├── ClientCard.tsx        # Used in recent clients widget
│   │   │   │   ├── ClientForm.tsx        # Shared form for create + edit
│   │   │   │   ├── ClientStatusBadge.tsx # active | paused | churned
│   │   │   │   ├── TokenManager.tsx      # Section inside ClientDetailPage:
│   │   │   │   │                         #   management token card + connection token card
│   │   │   │   ├── TokenRevealModal.tsx  # Shows plaintext token ONCE after generation
│   │   │   │   │                         #   Copy-to-clipboard, "I've saved this" confirm
│   │   │   │   ├── DeployStatusCard.tsx  # Frontend URL, backend URL, last health check,
│   │   │   │   │                         #   health badge, "ping now" button
│   │   │   │   └── SupabaseInfoCard.tsx  # Supabase ref, URL (no keys shown in UI)
│   │   │   ├── hooks/
│   │   │   │   ├── use-clients.ts            # useClients(), useClient(id), useCreateClient(),
│   │   │   │   │                             #   useUpdateClient()
│   │   │   │   └── use-client-tokens.ts      # useGenerateManagementToken(),
│   │   │   │                                 #   useRotateManagementToken(),
│   │   │   │                                 #   useGenerateConnectionToken(),
│   │   │   │                                 #   useRevokeConnectionToken(),
│   │   │   │                                 #   usePingDeployHealth()
│   │   │   ├── services/
│   │   │   │   └── clients.service.ts
│   │   │   └── types/
│   │   │       └── index.ts          # Client, CreateClientRequest, TokenResponse, etc.
│   │   │
│   │   ├── projects/
│   │   │   ├── pages/
│   │   │   │   ├── ProjectsPage.tsx      # All projects, filterable by status + client
│   │   │   │   └── ProjectDetailPage.tsx # Project info, status control, linked client
│   │   │   ├── components/
│   │   │   │   ├── ProjectsKanban.tsx    # Status columns: discovery→design→dev→review→live
│   │   │   │   ├── ProjectCard.tsx       # Compact card for kanban
│   │   │   │   ├── ProjectForm.tsx       # Create + edit
│   │   │   │   └── StatusSelect.tsx      # Status update control
│   │   │   ├── hooks/
│   │   │   │   └── use-projects.ts
│   │   │   └── services/
│   │   │       └── projects.service.ts
│   │   │
│   │   └── team/
│   │       ├── pages/
│   │       │   └── TeamPage.tsx          # Team list + invite + role management
│   │       ├── components/
│   │       │   ├── TeamTable.tsx         # Members with role badges, status, actions
│   │       │   ├── InviteModal.tsx       # Email input, role select, send invite
│   │       │   └── RoleAssignModal.tsx   # Direct role assignment (super_admin only)
│   │       ├── hooks/
│   │       │   └── use-team.ts           # useTeamMembers(), useInviteMember(),
│   │       │                             #   useAssignRole()
│   │       └── services/
│   │           └── team.service.ts
│   │
│   ├── lib/
│   │   ├── axios.ts                  # Axios instance + interceptors:
│   │   │                             #   - attach CSRF token to mutation requests
│   │   │                             #   - 401 → refresh → retry → redirect to /login
│   │   │                             #   - 429 → Sonner toast
│   │   │                             #   - 500 → silent slog (internal tool)
│   │   └── utils.ts                  # cn() (clsx + tailwind-merge), formatDate(),
│   │                                 #   formatRelative(), truncate()
│   │
│   ├── types/
│   │   └── index.ts                  # Shared interfaces used across features:
│   │                                 #   ApiError, PaginatedResponse<T>,
│   │                                 #   SortOrder, FilterParams
│   │
│   └── utils/
│       ├── errors.ts                 # parseApiError(err): extract message from axios error
│       └── format.ts                 # formatCurrency, formatDate, formatTokenPrefix
│
├── index.html
├── vite.config.ts
├── tailwind.config.ts
├── tsconfig.json
├── tsconfig.app.json
├── components.json                   # shadcn/ui config
├── .env.example                      # VITE_API_BASE_URL=http://localhost:8080/api
└── package.json
```

---

## Route Map

| Path | Component | Guard | Role |
|------|-----------|-------|------|
| `/login` | `LoginPage` | GuestRoute | — |
| `/dashboard` | `DashboardPage` | ProtectedRoute | all |
| `/clients` | `ClientsPage` | ProtectedRoute | all |
| `/clients/new` | `NewClientPage` | ProtectedRoute | admin+ |
| `/clients/:id` | `ClientDetailPage` | ProtectedRoute | admin+ |
| `/projects` | `ProjectsPage` | ProtectedRoute | all |
| `/projects/:id` | `ProjectDetailPage` | ProtectedRoute | all |
| `/team` | `TeamPage` | RoleProtectedRoute | super_admin |
| `*` | Redirect → `/dashboard` | — | — |

---

## Supabase Migrations Build Order

```sql
-- 001_create_users.sql
CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    role       TEXT NOT NULL CHECK (role IN ('super_admin','admin','moderator')),
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 002_create_role_requests.sql
CREATE TABLE role_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requested_by    UUID NOT NULL REFERENCES users(id),
    target_email    TEXT NOT NULL,
    requested_role  TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','approved','rejected')),
    reviewed_by     UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 003_create_clients.sql (abbreviated — full in master plan schema)
CREATE TABLE clients (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                        TEXT NOT NULL,
    business_name               TEXT,
    email                       TEXT NOT NULL,
    phone                       TEXT,
    plan_tier                   TEXT,
    status                      TEXT NOT NULL DEFAULT 'active'
                                CHECK (status IN ('active','paused','churned')),
    notes                       TEXT,
    management_token_hash       TEXT,
    management_token_prefix     TEXT,
    connection_token_hash       TEXT,
    connection_token_expires_at TIMESTAMPTZ,
    connection_token_used_at    TIMESTAMPTZ,
    client_supabase_project_ref TEXT,
    client_supabase_url         TEXT,
    claude_monthly_token_budget INTEGER NOT NULL DEFAULT 150000,
    claude_model                TEXT NOT NULL DEFAULT 'claude-haiku-4-5-20251001',
    portal_url                  TEXT,
    site_url                    TEXT,
    site_domain                 TEXT,
    domain_registrar            TEXT,
    dns_provider                TEXT,
    railway_service_url         TEXT,
    vercel_project_id           TEXT,
    created_by                  UUID REFERENCES users(id),
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 004 projects, 005 deploy_records, 006 audit_logs, 007 claude_usage
-- (defined in master plan schemas)
```

---

## Component Build Priority

Build in this order so each component has its dependencies ready:

```
Phase 1 — Shell
  AppLayout, AppSidebar, AppHeader, route guards, LoginPage + LoginForm

Phase 2 — Client Management (core feature)
  ClientsTable, ClientCard, ClientForm, ClientStatusBadge
  TokenRevealModal, TokenManager, DeployStatusCard

Phase 3 — Dashboard
  StatCard, StatsGrid, RecentClients, ProjectsOverview

Phase 4 — Projects
  ProjectCard, ProjectsKanban, ProjectForm, StatusSelect

Phase 5 — Team (least frequently used, build last)
  TeamTable, InviteModal, RoleAssignModal
```

---

## Config Files

**`.air.toml`** (hot reload):
```toml
[build]
  cmd = "go build -o ./tmp/main ./cmd/app"
  bin = "./tmp/main"
  include_ext = ["go"]
  exclude_dir = ["docs", "tmp", "vendor"]
[log]
  time = true
```

**`railway.toml`**:
```toml
[build]
  builder = "nixpacks"

[deploy]
  startCommand = "./app"
  healthcheckPath = "/health"
  healthcheckTimeout = 300
  restartPolicyType = "on_failure"
```
