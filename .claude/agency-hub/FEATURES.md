# agency-hub — Stage-by-Stage Feature Specification
> Reference this file at the start of every coding session.
> Every feature, validation rule, edge case, and acceptance criterion is documented here.
> Nothing ships until the acceptance criteria for that stage are met.

---

## Stage 1 — Project Foundation

### Go Backend Setup

**Module + dependencies:**
```bash
go mod init github.com/matts-org/agency-hub
go get github.com/go-chi/chi/v5
go get github.com/go-chi/chi/v5/middleware
go get github.com/go-playground/validator/v10
go get github.com/golang-jwt/jwt/v5
go get github.com/redis/go-redis/v9
go get github.com/resend/resend-go/v2
go get github.com/swaggo/swag/cmd/swag
go get github.com/swaggo/http-swagger
go get github.com/lib/pq
go get golang.org/x/crypto
```

**Config (`internal/config/config.go`):**
- Load every env var at startup using `os.Getenv`
- Required vars — `panic` with a clear message if missing:
  - `SUPABASE_DB_URL`, `JWT_SECRET`, `UPSTASH_REDIS_URL`, `UPSTASH_REDIS_TOKEN`, `RESEND_API_KEY`, `FRONTEND_URL`
- Optional with defaults: `PORT=8080`, `ENVIRONMENT=development`, `DB_SSLMODE=require`
- Exported `Config` struct — pass to all features via dependency injection, never call `os.Getenv` outside config

**Database (`internal/database/db.go`):**
- `Connect(cfg)` opens `database/sql` connection with `lib/pq`
- Enforce `sslmode=require` in production by appending to DSN if `ENVIRONMENT=production`
- Ping on startup — `log.Fatal` if unreachable
- `Close()` method for graceful shutdown

**Migration runner (`internal/database/migrate.go`):**
- Read all `.sql` files from `supabase/migrations/` in lexicographic order
- Execute each file contents as a single `db.Exec()` call
- Log each migration filename before + after execution
- If any migration fails, log the error and stop — do not run subsequent migrations
- `cmd/migrate/main.go` runs this and exits

**Logger (`pkg/logger/logger.go`):**
- In `development`: `slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})`
- In `production`: `slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})`
- Set as default: `slog.SetDefault(logger)`
- Call this in `main.go` before anything else

**Security middleware (`internal/middleware/security.go`):**
Set these headers on every response:
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Strict-Transport-Security: max-age=31536000; includeSubDomains (production only)
Content-Security-Policy: default-src 'self' (tighten per-feature as needed)
```

**Request logger middleware (`internal/middleware/logger.go`):**
- Wrap Chi's built-in `middleware.Logger` with slog output
- Log: method, path, status code, latency, request ID
- Use Chi's `middleware.RequestID` to inject request ID

**Health check:**
- `GET /health` — no auth required
- Returns `200` with:
```json
{ "status": "ok", "db": "ok", "redis": "ok", "version": "1.0.0" }
```
- Ping both DB and Redis before responding
- If either fails: return `503` with the failing service name

**`main.go` startup sequence:**
```
1. Init logger
2. Load config (panic if invalid)
3. Connect DB (fatal if fail)
4. Connect Redis (fatal if fail)
5. Run pending migrations
6. Init all repositories + services + handlers
7. Build Chi router, register all routes
8. Serve on cfg.Port
```

**`cmd/admin/main.go` — Bootstrap CLI:**
- `promote --email <email> --role <role>` subcommand
- Fetches user by email from DB
- Updates their role to the specified value
- Only works for `super_admin`, `admin`, `moderator` roles
- Logs success or error
- This is the ONLY way to set the first `super_admin`

### Acceptance Criteria — Stage 1
- [ ] `go run cmd/app/main.go` starts without errors
- [ ] `GET /health` returns `{ "status": "ok", "db": "ok", "redis": "ok" }`
- [ ] `go run cmd/migrate/main.go` runs all SQL files without error
- [ ] Missing required env var causes a clear panic message naming the missing variable
- [ ] Security headers present on all responses (verify with `curl -I`)

---

## Stage 2 — Authentication System

### Login (`POST /api/auth/login`)

**Request validation:**
```go
type LoginRequest struct {
    Email    string `json:"email"    validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}
```

**Flow:**
1. Validate request body
2. Check account lockout: `GET redis key: lockout:{email}` — if exists, return `429` with message "Account temporarily locked. Try again in 15 minutes."
3. Fetch user by email from DB — if not found, return `401` ("Invalid credentials" — do NOT reveal whether email exists)
4. Check `user.is_active` — if false, return `401` ("Account is inactive")
5. Verify password with `bcrypt.CompareHashAndPassword`
6. If password wrong: increment Redis counter `login_attempts:{email}` (expire 15 min). At 5 → set `lockout:{email}` key (15 min TTL), reset attempts counter. Return `401`
7. If password correct: reset login attempts counter
8. Generate JWT access token (15 min) + refresh token (random 32-byte hex, store SHA-256 hash in DB with 7-day expiry)
9. Set both as HTTP-only, Secure, SameSite=Strict cookies
10. Write audit log: action=`LOGIN`, entity_type=`user`, entity_id=user.ID
11. Return `200` with user profile (id, email, name, role)

**JWT claims:**
```go
type Claims struct {
    UserID   string `json:"user_id"`
    Email    string `json:"email"`
    Role     string `json:"role"`
    IsActive bool   `json:"is_active"`
    jwt.RegisteredClaims
}
```
- Algorithm: HS256
- Access token expiry: 15 minutes
- Issuer: `"agency-hub"`

**Refresh token storage:**
```sql
-- Stored in Redis (not DB) for fast lookup:
-- Key: refresh_token:{token_hash}
-- Value: user_id (JSON)
-- TTL: 7 days
```

### Logout (`POST /api/auth/logout`)
- Requires auth middleware (valid access token)
- Extract refresh token from cookie
- Delete `refresh_token:{hash}` from Redis
- Clear both cookies (set Max-Age=0)
- Write audit log: action=`LOGOUT`
- Return `200`

### Refresh (`POST /api/auth/refresh`)
- Extract refresh token from cookie — if missing, return `401`
- Hash it, look up in Redis — if not found or expired, return `401`
- Fetch user from DB — if inactive, return `401`
- **Rotation:** delete old Redis key, generate new refresh token, store new hash in Redis
- Issue new access token + new refresh token cookies
- Return `200` with new access token claims

### CSRF (`GET /api/auth/csrf`)
- No auth required
- Generate a random 32-byte token
- Store in Redis: `csrf:{token_hash}` with 1-hour TTL, value = `"valid"`
- Return token in response body: `{ "csrf_token": "..." }`
- Client stores this and sends as `X-CSRF-Token` header on all mutations

**CSRF middleware (`internal/middleware/csrf.go`):**
- On every POST/PUT/PATCH/DELETE:
  - Read `X-CSRF-Token` header
  - Hash it, check Redis key exists
  - If not: return `403` ("Invalid or missing CSRF token")
  - CSRF tokens are NOT rotated per request (session-scoped, 1hr TTL)
  - Exempt: `GET /api/validate-management-token` (API-to-API)

### Profile (`GET /api/auth/profile`)
- Requires auth middleware
- Fetch fresh user from DB using `claims.UserID`
- Return: id, email, name, role, is_active, created_at
- Do NOT return password hash

### Rate Limiting (`internal/middleware/ratelimit.go`)

Sliding window implementation using Redis sorted sets:
```go
func NewRateLimiter(rdb *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler
```
- Key pattern: `ratelimit:{route_identifier}:{ip_address}`
- Algorithm:
  1. `ZREMRANGEBYSCORE key 0 (now - window_seconds)`
  2. `ZCARD key` — if >= limit, return `429`
  3. `ZADD key now now`
  4. `EXPIRE key window_seconds*2`
- On Redis failure: fail open (allow request, log error)

**Applied limits:**
- Auth routes (login, register): 5 requests/minute per IP
- Token refresh: 15 requests/minute per IP
- Authenticated routes: 30 requests/minute per user_id (not IP)
- `GET /validate-management-token`: 10 requests/minute per IP

### Password Hashing
- Use `bcrypt` with cost factor 12
- `HashPassword(plain string) (string, error)` in `internal/utils/crypto.go`
- `CheckPassword(plain, hash string) bool`

### Acceptance Criteria — Stage 2
- [ ] Login with correct credentials → JWT + refresh token cookies set
- [ ] Login with wrong password 5× → account locked for 15 min
- [ ] Locked account cannot login even with correct password
- [ ] Logout → cookies cleared, refresh token invalidated
- [ ] Refresh → new tokens issued, old refresh token no longer valid
- [ ] CSRF token required on POST routes — request without it gets 403
- [ ] Rate limit on login: 6th request in 1 min → 429
- [ ] Profile returns correct user data from DB (not just claims)

---

## Stage 3 — Client Management

### Create Client (`POST /api/clients`)
**Required roles:** admin, super_admin

**Request:**
```go
type CreateClientRequest struct {
    Name         string `json:"name"          validate:"required,min=2,max=100"`
    BusinessName string `json:"business_name" validate:"max=150"`
    Email        string `json:"email"         validate:"required,email"`
    Phone        string `json:"phone"         validate:"max=30"`
    PlanTier     string `json:"plan_tier"     validate:"oneof=basic standard premium"`
    Notes        string `json:"notes"         validate:"max=2000"`
}
```

**Flow:**
1. Validate request
2. Check email uniqueness in clients table — `409` if already exists
3. Insert client record with `status=active`
4. **Immediately generate management_token:**
   - `crypto/rand` 32 bytes → hex string (64 chars)
   - `management_token_prefix` = first 8 chars (shown in UI for identification)
   - `management_token_hash` = SHA-256 hex of full token
   - Store hash + prefix in DB
5. Write audit log: `CREATE_CLIENT`
6. Return `201` with full client record **including plaintext management_token** (this is the ONLY time it is returned)

**Response includes:**
```go
type CreateClientResponse struct {
    Client          ClientResponse `json:"client"`
    ManagementToken string         `json:"management_token"` // plaintext, shown ONCE
}
```

### List Clients (`GET /api/clients`)
**Required roles:** all authenticated

**Query params:**
- `status` — filter by `active|paused|churned` (optional)
- `page` — default 1
- `limit` — default 20, max 100
- `search` — partial match on name or business_name (ILIKE)

**Response:**
```go
type PaginatedClientsResponse struct {
    Clients    []ClientResponse `json:"clients"`
    Total      int              `json:"total"`
    Page       int              `json:"page"`
    TotalPages int              `json:"total_pages"`
}
```

### Get Client (`GET /api/clients/:id`)
**Required roles:** admin, super_admin

Returns full client record. Does NOT return token hashes. Shows `management_token_prefix` only.

### Update Client (`PUT /api/clients/:id`)
**Required roles:** admin, super_admin

All fields optional (partial update). Validate same rules as create. Write audit log.

Do NOT allow updating token fields via this endpoint — tokens have dedicated endpoints.

### Management Token Endpoints

**Generate (on create — built into POST /api/clients). Rotate (`POST /api/clients/:id/management-token/rotate`):**
- Required role: super_admin only
- Generate new 32-byte token same way as create
- Update hash + prefix in DB
- Old token is immediately invalidated (hash replaced)
- Write audit log: `ROTATE_MANAGEMENT_TOKEN`
- Return plaintext new token ONCE

**Validate (`GET /api/validate-management-token`):**
- Auth: `Authorization: Bearer <management_token>` header + `X-Client-ID` header
- NO JWT required, NO CSRF required — this is called by external services
- Rate limit: 10/min per IP
- Flow:
  1. Extract Bearer token from header
  2. SHA-256 hash it
  3. Fetch client by `X-Client-ID`
  4. Compare `management_token_hash` — if no match: `401`
  5. Check `client.status == "active"` — if not: `403` ("Client is not active")
  6. Return: `{ "client_id": "...", "client_name": "...", "status": "active" }`
- Do NOT log this endpoint (called on every startup — too noisy). Log only on failure.

### Connection Token Endpoints

**Generate (`POST /api/clients/:id/connection-token`):**
- Required role: admin, super_admin
- Generate 32-byte random hex token
- Store `connection_token_hash` (SHA-256) + `connection_token_expires_at` (7 days from now) in DB
- Clears any previous unused connection_token
- Write audit log: `GENERATE_CONNECTION_TOKEN`
- Return plaintext token ONCE: `{ "connection_token": "...", "expires_at": "..." }`

**Revoke (`DELETE /api/clients/:id/connection-token`):**
- Required role: admin, super_admin
- Set `connection_token_hash = NULL`, `connection_token_expires_at = NULL` in DB
- Write audit log: `REVOKE_CONNECTION_TOKEN`

**Validate (`POST /api/validate-connection-token`):**
- Auth: `Authorization: Bearer <management_token>` + `X-Client-ID`
- Body: `{ "token": "<connection_token>", "email": "<user_email>" }`
- Flow:
  1. Validate management token (same as above)
  2. Hash the connection_token, compare to `connection_token_hash`
  3. Check not expired: `connection_token_expires_at > NOW()`
  4. Check not already used: `connection_token_used_at IS NULL`
  5. If all valid: set `connection_token_used_at = NOW()` in DB
  6. Return: `{ "valid": true, "client_id": "...", "client_name": "..." }`
  7. On any failure: `{ "valid": false, "reason": "expired|used|invalid" }`
- Write audit log: `CONNECTION_TOKEN_VALIDATED`

### Deploy Record Endpoints

**Upsert (`PUT /api/clients/:id/deploy`):**
```go
type UpsertDeployRequest struct {
    FrontendURL     string `json:"frontend_url"`
    BackendURL      string `json:"backend_url"`
    SupabaseRef     string `json:"supabase_ref"`
    VercelProjectID string `json:"vercel_project_id"`
    RailwayURL      string `json:"railway_service_url"`
}
```
- INSERT OR UPDATE deploy_records for this client_id
- Also updates matching fields on the `clients` table (site_url, railway_service_url, vercel_project_id)

**Ping (`POST /api/clients/:id/deploy/ping`):**
- Required role: admin, super_admin
- Fetch client's `site_url` from DB — if empty, return `400` ("No site URL configured")
- HTTP GET to `{site_url}/health` with 5-second timeout
- Update `deploy_records.last_checked_at = NOW()`
- Update `health_status`:
  - 200 response → `healthy`
  - Non-200 response → `degraded`
  - Timeout or connection refused → `down`
- Return result: `{ "health_status": "healthy", "checked_at": "..." }`

### Acceptance Criteria — Stage 3
- [ ] POST /api/clients returns 201 with management_token in plaintext exactly once
- [ ] Second GET /api/clients/:id does NOT return management_token — only prefix
- [ ] Validate management token: valid → 200, wrong token → 401, inactive client → 403
- [ ] Connection token: generate → validate (marks used) → validate again → 400 "used"
- [ ] Connection token expiry: manually set expires_at to past → validate → 400 "expired"
- [ ] Ping: valid URL responds 200 → health_status = "healthy"
- [ ] Ping: unreachable URL → health_status = "down"

---

## Stage 4 — Team Management & RBAC

### Team Members

**List (`GET /api/team`):**
- Required role: super_admin
- Returns all users: id, email, name, role, is_active, created_at
- Ordered by created_at DESC

**Invite (`POST /api/team/invite`):**
- Required role: admin, super_admin
```go
type InviteRequest struct {
    Email string `json:"email" validate:"required,email"`
    Name  string `json:"name"  validate:"required,min=2,max=100"`
    Role  string `json:"role"  validate:"required,oneof=admin moderator"`
}
```
- Check email not already in users table — `409` if exists
- Create user record with a temporary random password (they'll reset on first login via forgot password — or add magic link later)
- Send invite email via Resend: "You've been invited to [Agency Name]'s management platform"
- Email contains a login link (just the URL, no magic — they set their own password)
- Write audit log: `INVITE_TEAM_MEMBER`

**Note:** Only `admin` and `moderator` roles can be assigned via invite. `super_admin` only via bootstrap CLI or direct assignment.

### Role Management

**Direct assign (`POST /api/team/roles/assign`):**
- Required role: super_admin ONLY
```go
type AssignRoleRequest struct {
    UserID string `json:"user_id" validate:"required,uuid"`
    Role   string `json:"role"    validate:"required,oneof=super_admin admin moderator"`
}
```
- Update `users.role` for target user
- Cannot demote or change your own role
- Write audit log: `ASSIGN_ROLE`, metadata includes old_role + new_role

**Request promotion (`POST /api/team/roles/requests`):**
- Required role: admin, moderator
```go
type RoleRequestBody struct {
    TargetEmail   string `json:"target_email"   validate:"required,email"`
    RequestedRole string `json:"requested_role" validate:"required,oneof=admin super_admin"`
}
```
- Creates a `role_requests` record with `status=pending`
- Cannot request a role equal to or lower than current role
- Cannot create duplicate pending requests for the same email+role
- Sends email to all `super_admin` users: "A role promotion has been requested"
- Return `201`

**Approve (`POST /api/team/roles/requests/:id/approve`):**
- Required role: super_admin
- Fetch request — if not found: `404`, if not `pending`: `400`
- Look up user by `request.target_email` — if not found: `404`
- Update `users.role` to `request.requested_role`
- Update `role_requests.status = approved`, `reviewed_by = current user`
- Send email to the promoted user
- Write audit log: `APPROVE_ROLE_REQUEST`

**Reject (`POST /api/team/roles/requests/:id/reject`):**
- Required role: super_admin
```go
type RejectRequest struct {
    Reason string `json:"reason" validate:"required,min=5,max=500"`
}
```
- Update `role_requests.status = rejected`, `reviewed_by = current user`
- Send rejection email to the requester with reason
- Write audit log: `REJECT_ROLE_REQUEST`

### Audit Logging (`internal/audit/service.go`)

`LogAction` is called by all services for any data-modifying operation:
```go
func (s *Service) LogAction(ctx context.Context, actorID, action, entityType, entityID string, metadata map[string]any) error
```
- Extracts IP from context (set by request logger middleware)
- Serializes metadata to JSONB
- Inserts into `audit_logs` table
- Non-blocking: log the error if DB write fails, do not fail the parent request

**Tracked actions (constants):**
```
LOGIN, LOGOUT, INVITE_TEAM_MEMBER, ASSIGN_ROLE,
REQUEST_ROLE_PROMOTION, APPROVE_ROLE_REQUEST, REJECT_ROLE_REQUEST,
CREATE_CLIENT, UPDATE_CLIENT, GENERATE_MANAGEMENT_TOKEN,
ROTATE_MANAGEMENT_TOKEN, GENERATE_CONNECTION_TOKEN,
REVOKE_CONNECTION_TOKEN, CONNECTION_TOKEN_VALIDATED,
CREATE_PROJECT, UPDATE_PROJECT, UPDATE_PROJECT_STATUS,
UPSERT_DEPLOY_RECORD
```

### Mailer (`internal/mailer/service.go`)

All emails sent via Resend HTTP API (`github.com/resend/resend-go/v2`):

**Email types:**
- `SendTeamInvite(to, name, loginURL string)`
- `SendRoleApproved(to, name, newRole string)`
- `SendRoleRejected(to, name, reason string)`
- `SendBudgetAlert(adminEmails []string, clientName string, percentUsed int)` (for Claude usage)

All emails use simple HTML templates (inline, no template files needed at this stage). Subject lines:
- Invite: `"You've been invited to [Agency] Management"`
- Approved: `"Your role has been updated"`
- Rejected: `"Role request update"`
- Budget: `"[Client Name] has used {N}% of their Claude budget"`

### Acceptance Criteria — Stage 4
- [ ] Invite creates user + sends email (verify Resend receives the call)
- [ ] super_admin can directly assign any role including super_admin
- [ ] admin cannot call /team/roles/assign → 403
- [ ] Role request → approve flow: user's role actually updated in DB
- [ ] Role request → reject: status = rejected, reason stored
- [ ] Every modifying action creates an audit_logs row

---

## Stage 5 — Projects & Deploy Tracking

### Projects

**Create (`POST /api/projects`):**
```go
type CreateProjectRequest struct {
    ClientID          string `json:"client_id"           validate:"required,uuid"`
    Name              string `json:"name"                validate:"required,min=2,max=150"`
    Description       string `json:"description"         validate:"max=1000"`
    Status            string `json:"status"              validate:"oneof=discovery design development review live maintenance"`
    LinearProjectURL  string `json:"linear_project_url"  validate:"omitempty,url"`
    NotionPageURL     string `json:"notion_page_url"     validate:"omitempty,url"`
    EstimatedDelivery string `json:"estimated_delivery"  validate:"omitempty,datetime=2006-01-02"`
    AssignedTo        string `json:"assigned_to"         validate:"omitempty,uuid"`
}
```
- Verify `client_id` exists — `404` if not
- Verify `assigned_to` (if provided) is a valid user ID
- Default `status = "discovery"` if not provided
- Write audit log: `CREATE_PROJECT`

**List (`GET /api/projects`):**
- Query params: `client_id` (filter), `status` (filter), `page`, `limit`
- All authenticated users

**Get by client (`GET /api/clients/:id/projects`):**
- Returns all projects for a specific client
- All authenticated users

**Update status (`PATCH /api/projects/:id/status`):**
```go
type UpdateStatusRequest struct {
    Status string `json:"status" validate:"required,oneof=discovery design development review live maintenance"`
}
```
- Write audit log: `UPDATE_PROJECT_STATUS`, metadata = `{ "old_status": "...", "new_status": "..." }`

### Dashboard Aggregate (`GET /api/dashboard`)

**Response:**
```go
type DashboardStats struct {
    ActiveClients    int              `json:"active_clients"`
    TotalClients     int              `json:"total_clients"`
    ProjectsByStatus map[string]int   `json:"projects_by_status"` // {"live": 3, "development": 2, ...}
    RecentClients    []ClientSummary  `json:"recent_clients"`      // last 5
    ClaudeUsageThisMonth int          `json:"claude_usage_this_month"` // total tokens across all clients
    LiveSites        int              `json:"live_sites"`           // clients with health_status=healthy
}
```

All counts from a single query per data type. No N+1 queries — use SQL COUNT + GROUP BY.

### Acceptance Criteria — Stage 5
- [ ] Project created with valid client_id → 201
- [ ] Project creation with non-existent client_id → 404
- [ ] Status update with invalid status → 400
- [ ] GET /api/dashboard returns all fields with correct counts
- [ ] GET /api/clients/:id/projects returns only that client's projects

---

## Stage 6 — Frontend Foundation

### React Project Setup (`web/`)

```bash
pnpm create vite@latest web -- --template react-ts
cd web
pnpm add @tanstack/react-query@5 react-router-dom@7 axios
pnpm add framer-motion sonner lucide-react
pnpm add react-hook-form zod @hookform/resolvers
pnpm add -D tailwindcss @tailwindcss/vite
pnpm dlx shadcn@latest init
```

**shadcn/ui components to install upfront:**
```bash
pnpm dlx shadcn@latest add button card input label badge table dialog sheet
pnpm dlx shadcn@latest add dropdown-menu avatar skeleton separator alert
pnpm dlx shadcn@latest add select textarea form tabs popover tooltip progress
pnpm dlx shadcn@latest add alert-dialog command
```

### Axios Client (`src/lib/axios.ts`)

```typescript
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  withCredentials: true,  // CRITICAL — sends cookies cross-origin
})
```

**CSRF interceptor:**
- On app load: call `GET /api/auth/csrf`, store token in module-level variable
- Request interceptor: for POST/PUT/PATCH/DELETE, add `X-CSRF-Token` header
- If CSRF token is null (not yet fetched), fetch it first then retry

**401 interceptor:**
- On 401 response: attempt `POST /api/auth/refresh`
- If refresh succeeds: retry original request once
- If refresh fails (401 again): clear local auth state, redirect to `/login`
- Use a flag to prevent infinite refresh loops

**Other interceptors:**
- 429: `toast.error("Too many requests. Please slow down.")` via Sonner
- Network error: `toast.error("Connection error. Check your internet.")`
- 500: silent — log to console only (internal tool, don't alarm users)

### Auth Context (`src/contexts/auth-context.tsx`)

```typescript
interface AuthContextValue {
  user: User | null
  isLoading: boolean
  isAuthenticated: boolean
  logout: () => Promise<void>
}
```

- Backed by `useQuery({ queryKey: ['auth', 'profile'], queryFn: authService.getProfile })`
- `staleTime: 5 * 60 * 1000` (5 min — don't re-fetch profile on every focus)
- `retry: false` — on 401, don't retry
- On query error (401): set user to null
- `logout()`: call `POST /api/auth/logout`, then `queryClient.clear()`

### Route Guards

**`ProtectedRoute`:** If `!isAuthenticated && !isLoading` → redirect to `/login`

**`GuestRoute`:** If `isAuthenticated` → redirect to `/dashboard`

**`RoleProtectedRoute({ roles: string[] })`:** If `!roles.includes(user.role)` → redirect to `/dashboard` with a toast "Access denied"

### Central Route Registry (`src/config/routes.tsx`)

All page components lazy-loaded with `React.lazy()`. Wrapped in `Suspense` with a skeleton fallback. Structure:

```typescript
{
  path: '/login',
  element: <GuestRoute><Suspense><LazyLoginPage /></Suspense></GuestRoute>
}
{
  path: '/',
  element: <ProtectedRoute><AppLayout /></ProtectedRoute>,
  children: [
    { path: 'dashboard', element: <Suspense><LazyDashboard /></Suspense> },
    // ... all authenticated routes as children of AppLayout
  ]
}
```

### AppLayout + Sidebar

**AppSidebar nav items (role-aware visibility):**
```
Dashboard      → /dashboard       (all roles)
Clients        → /clients         (all roles)
Projects       → /projects        (all roles)
Team           → /team            (super_admin only)
```

**Footer of sidebar:** User avatar (initials fallback), name, role badge, logout button

**Active route:** Use `useLocation()` to highlight current route

**Responsive:** Sidebar collapses on mobile. Use shadcn `Sheet` for mobile nav drawer.

### Acceptance Criteria — Stage 6
- [ ] App loads, calls `GET /api/auth/csrf` once on init
- [ ] Unauthenticated user visiting `/dashboard` → redirected to `/login`
- [ ] Authenticated user visiting `/login` → redirected to `/dashboard`
- [ ] Moderator visiting `/team` → redirected with "Access denied" toast
- [ ] 401 on any request → refresh attempted → on failure → redirect to /login
- [ ] AppSidebar shows "Team" link only for super_admin

---

## Stage 7 — Auth UI

### Login Page (`src/features/auth/pages/LoginPage.tsx`)

Full-page layout (NOT inside AppLayout). Centered card.

**Form fields:**
- Email (type="email", required)
- Password (type="password", required, min 8 chars client-side)
- Submit button with loading state (disabled + spinner while pending)

**Zod schema:**
```typescript
const schema = z.object({
  email: z.string().email("Enter a valid email"),
  password: z.string().min(8, "Password must be at least 8 characters"),
})
```

**Error states to handle:**
- Invalid credentials → display "Invalid email or password" below form (not toast)
- Account locked → display "Account locked. Try again in 15 minutes." below form
- Account inactive → display "Your account has been deactivated. Contact your administrator."
- Network error → Sonner toast "Connection error"
- All API error messages come from the response body `{ "error": "..." }` field

**Post-login:** Call `queryClient.invalidateQueries(['auth', 'profile'])` then navigate to `/dashboard`

### Acceptance Criteria — Stage 7
- [ ] Valid login → dashboard
- [ ] Invalid credentials → inline error (not toast, not page reload)
- [ ] Account locked message shown correctly
- [ ] Submit button disabled during loading
- [ ] Email field auto-focused on page load

---

## Stage 8 — Client Management UI

### Clients List Page

**Features:**
- Searchable by name/business name (debounced 300ms, client-side filter or query param)
- Filter by status: All | Active | Paused | Churned (tab bar or select)
- Sortable columns: Name, Plan Tier, Status, Created At
- "New Client" button → `/clients/new` (admin+ only — hide for moderators)
- Pagination (if > 20 clients)
- Each row: name, business name, status badge, plan tier, created date, "View →" link

**Empty state:** "No clients yet. Create your first client to get started."

**Status badge colors:**
- `active` → green
- `paused` → yellow
- `churned` → gray

### Client Detail Page

Tabbed layout or stacked sections. Sections:

**Profile section:**
- Editable fields (admin+): name, business name, email, phone, plan tier, notes
- Inline edit (click to edit, save/cancel) OR edit modal
- Status change: Active / Paused / Churned select (admin+)

**Token Manager section (admin+ only):**

*Management Token card:*
- Shows: prefix (first 8 chars) + `...` — never full token
- Shows: "Created [date]"
- Button: "Rotate Token" → confirmation dialog ("This will break any deployed portal/site using the current token. Are you sure?") → calls rotate endpoint → opens `TokenRevealModal`

*Connection Token card:*
- Shows: status (Active / Used / Expired / None)
- Shows: expiry date if active
- Button: "Generate New Token" → calls generate endpoint → opens `TokenRevealModal`
- Button: "Revoke" (only if active, not used) → confirmation dialog

**`TokenRevealModal`:**
- Title: "Save this token — it won't be shown again"
- Displays token in monospace code block
- Copy-to-clipboard button (copies + shows checkmark)
- "I've saved this token" button (required to close — cannot close by clicking outside)
- Warning banner: "This token will not be accessible after closing this dialog"

**Deploy Status card:**
- Frontend URL, Backend URL, Supabase ref (all editable)
- Health status badge: Healthy 🟢 / Degraded 🟡 / Down 🔴 / Unknown ⚪
- Last checked: relative time ("2 hours ago")
- "Ping Now" button → calls ping endpoint → updates badge in place

**Projects section:**
- Mini list of this client's projects with status + name
- "View all projects →" link

### New Client Page

Multi-field form:
```
Name*          Business Name
Email*         Phone
Plan Tier*     Notes (textarea)
```

On submit:
1. POST /api/clients
2. Response contains `management_token` → **immediately open TokenRevealModal**
3. After modal closed → navigate to `/clients/:id`

### Acceptance Criteria — Stage 8
- [ ] Token reveal modal cannot be closed without clicking "I've saved this token"
- [ ] Management token only shows prefix in the UI (not full token)
- [ ] New client form opens TokenRevealModal immediately after creation
- [ ] Moderator: cannot see "New Client" button or token management section
- [ ] Ping updates health badge without page reload
- [ ] Rotate token shows confirmation dialog before proceeding

---

## Stage 9 — Projects UI

### Projects Page (Kanban View)

6 columns: Discovery | Design | Development | Review | Live | Maintenance

Each project card shows: client name (badge), project name, assigned to (avatar), due date

**Filtering:** By client (select dropdown of all clients) + by assignee

**Project card actions:** Click → go to `/projects/:id`

**"New Project" button** (admin+): opens a modal form (not a new page)

### Project Detail Page

- Client link (go to client detail)
- Status select (update inline)
- Linear project URL + Notion page URL (clickable links, editable)
- Assigned to (team member select)
- Estimated vs actual delivery dates
- Description (editable textarea)
- Edit triggers `PATCH` to correct endpoints

### Acceptance Criteria — Stage 9
- [ ] Projects appear in correct kanban column
- [ ] Status change updates the column in real-time (optimistic update or refetch)
- [ ] Project detail links correctly to client

---

## Stage 10 — Team UI

### Team Page (super_admin only)

Table columns: Avatar + Name, Email, Role (badge), Status (Active/Inactive), Joined, Actions

**Actions per row:**
- Change role (opens RoleAssignModal — super_admin only)

**"Invite Member" button:** Opens InviteModal
- Fields: Name, Email, Role (admin / moderator only)
- Success: toast "Invite sent to [email]", row appears in table with "Invited" status

**Role requests section (below team table):**
- Pending requests: target email, requested role, requested by, date
- Approve / Reject buttons (super_admin only)
- Rejection requires a reason (textarea in a modal)

### Acceptance Criteria — Stage 10
- [ ] Invite → user appears in table
- [ ] Role change reflected immediately in the table
- [ ] Pending role requests visible and actionable
- [ ] Page itself not accessible by admin or moderator (404 or redirect)

---

## Stage 11 — Dashboard UI

4 stat cards (top row):
- Active Clients (count, trend optional)
- Live Sites (count with Healthy health_status)
- Open Projects (count of discovery+design+development+review statuses)
- Claude Usage This Month (total tokens, formatted: "12.4k tokens")

Recent Clients (last 5):
- Client name, business name, status badge, "View →" link

Projects by Status:
- Simple list or horizontal bar for each status with count
- Highlight "live" in green

All data from `GET /api/dashboard` — single request.

### Acceptance Criteria — Stage 11
- [ ] Dashboard loads with a single API call
- [ ] All 4 stats display correctly
- [ ] Skeleton loading state shown while fetching

---

## Stage 12 — QA & Deploy

### Pre-Deploy Checklist
- [ ] All env vars documented in `.env.example` with descriptions
- [ ] No hardcoded secrets, URLs, or credentials in code
- [ ] All Go features have Swagger annotations — `swag init` runs without warnings
- [ ] `go vet ./...` passes clean
- [ ] All routes return consistent JSON error format `{ "error": "..." }`
- [ ] Rate limiting verified on auth routes (manual test or simple script)
- [ ] RBAC: test every role against every restricted endpoint
- [ ] Token reveal modal flow tested end-to-end
- [ ] Audit logs populated for every tracked action
- [ ] Frontend: no `console.error` on normal user flows
- [ ] Frontend: all loading + error + empty states implemented (no undefined render)

### Railway Deployment
- Set all env vars in Railway dashboard (not via config file)
- `railway.toml` healthcheck points to `/health`
- Verify `/health` returns 200 after deploy
- Verify Swagger UI accessible at `/swagger/index.html`

### Vercel Deployment
- `VITE_API_BASE_URL` set to Railway URL
- Verify login works (CORS allowing Vercel domain)
- Check that `withCredentials: true` works cross-origin (CORS must include `AllowCredentials(true)` and specific origin, not `*`)
