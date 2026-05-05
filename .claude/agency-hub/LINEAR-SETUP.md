# agency-hub — Linear Project Setup
> Copy this into Linear before writing a single line of code.

---

## Workspace Setup

```
Linear Workspace: [Matt's agency name]
Team name:        agency-hub
Team identifier:  AH  (issues will be: AH-1, AH-2, ...)
```

---

## Labels

Create these labels in the team settings:

| Label | Color | Use for |
|-------|-------|---------|
| `backend` | Blue `#3B82F6` | Go API work |
| `frontend` | Purple `#8B5CF6` | React UI work |
| `design` | Pink `#EC4899` | Variants design work |
| `infra` | Orange `#F97316` | DB, Redis, Railway, Vercel setup |
| `security` | Red `#EF4444` | Auth, tokens, RBAC, rate limiting |
| `bug` | Red `#DC2626` | Something broken |
| `chore` | Gray `#6B7280` | Config, tooling, cleanup |

---

## Milestones (Epics)

Create these as **Milestones** in Linear:

```
M1: Foundation & Auth
M2: Client Management
M3: Team Management
M4: Projects & Deploy Tracking
M5: Dashboard & Analytics
M6: Frontend Shell
M7: Client UI
M8: Projects UI
M9: Team UI
M10: Dashboard UI
M11: QA & Launch
```

---

## Cycles (Sprints)

```
Cycle 1:  M1 — Foundation & Auth (backend)
Cycle 2:  M2 — Client Management (backend)
Cycle 3:  M3 + M4 — Team + Deploy (backend)
Cycle 4:  M6 + M7 — Frontend Shell + Client UI
Cycle 5:  M8 + M9 + M10 — Remaining UI
Cycle 6:  M11 — QA, polish, launch
```

---

## All Issues

Format: **[ID suggestion] Title** | Labels | Priority | Milestone | Notes

---

### M1 — Foundation & Auth

**AH-1** Set up Go project structure
`backend` `chore` · Priority: Urgent · M1
- Init Go module, install Chi v5, validator, swag, air, slog
- Create cmd/app, cmd/admin, cmd/migrate folder structure
- Create internal/ feature folders (empty)
- Set up .env.example, .air.toml, railway.toml, go.mod

**AH-2** Set up Supabase project (agency)
`infra` · Priority: Urgent · M1
- Create Supabase project: agency-hub-prod
- Save credentials to 1Password
- Run 001_create_users.sql migration

**AH-3** Implement config loader
`backend` · Priority: Urgent · M1
- internal/config/config.go
- Load all env vars, validate required ones, return typed Config
- Panic with clear message if required var is missing

**AH-4** Implement database connection + migration runner
`backend` `infra` · Priority: Urgent · M1
- internal/database/db.go — sql.Open + ping
- internal/database/migrate.go — ordered SQL file runner
- cmd/migrate/main.go — CLI entrypoint

**AH-5** Implement security middleware stack
`backend` `security` · Priority: Urgent · M1
- internal/middleware/security.go — all security headers
- internal/middleware/logger.go — request logging with slog
- Wire both as global middleware in main.go

**AH-6** Implement JWT auth system
`backend` `security` · Priority: Urgent · M1
- internal/auth/model.go, repository.go, service.go, handler.go, routes.go
- Login endpoint (email + password, account lockout after 5 fails)
- Logout (invalidate refresh token)
- Refresh endpoint (rotate refresh token)
- GET /auth/csrf (generate CSRF token)
- GET /auth/profile (requires auth)
- Depends on: AH-1, AH-3, AH-4

**AH-7** Implement rate limiting middleware
`backend` `security` · Priority: High · M1
- internal/middleware/ratelimit.go
- Upstash Redis sliding window
- Tiered: 5/min auth routes, 30/min authenticated routes
- Depends on: AH-3

**AH-8** Implement CSRF middleware
`backend` `security` · Priority: High · M1
- internal/middleware/csrf.go
- Validate X-CSRF-Token header on all mutation requests
- Depends on: AH-6

**AH-9** Implement JWT extraction middleware
`backend` `security` · Priority: High · M1
- internal/middleware/auth.go — extract + validate JWT, inject Claims into ctx
- internal/middleware/rbac.go — RequireRole() factory
- Depends on: AH-6

**AH-10** Set up Swagger
`backend` `chore` · Priority: Medium · M1
- Install swag, run swag init
- Wire swagger UI route in main.go
- Add Health endpoint with annotation: GET /health

**AH-11** Bootstrap CLI (first super_admin)
`backend` `security` · Priority: High · M1
- cmd/admin/main.go
- `promote --email x --role super_admin` command
- Depends on: AH-4, AH-6

**AH-12** Run all migrations + verify Supabase schema
`infra` · Priority: High · M1
- Run all 7 migration files
- Verify in Supabase dashboard
- Depends on: AH-4, AH-2

---

### M2 — Client Management (Backend)

**AH-13** Implement client CRUD
`backend` · Priority: Urgent · M2
- internal/client/model.go, repository.go, service.go, handler.go, routes.go
- GET /clients (all, with pagination + filter by status)
- POST /clients (create — auto-generates management_token)
- GET /clients/:id
- PUT /clients/:id
- Audit log every create + update
- Depends on: AH-9, AH-8

**AH-14** Implement management token generation + rotation
`backend` `security` · Priority: Urgent · M2
- POST /clients/:id/management-token (generate, returns plaintext ONCE)
- POST /clients/:id/management-token/rotate
- GET /validate-management-token (no CSRF — API-to-API endpoint)
- Token: 32-byte crypto/rand → hex, stored SHA-256 hashed
- Depends on: AH-13

**AH-15** Implement connection token (client user onboarding)
`backend` `security` · Priority: Urgent · M2
- POST /clients/:id/connection-token (generate, 7-day expiry)
- DELETE /clients/:id/connection-token (revoke)
- POST /validate-connection-token (called by portal backend with management_token auth)
- Depends on: AH-13, AH-14

**AH-16** Implement deploy record management
`backend` · Priority: High · M4
- internal/deploy/ — all files
- PUT /clients/:id/deploy (update URLs + refs)
- GET /clients/:id/deploy
- POST /clients/:id/deploy/ping (ping client site /health, update health_status)
- Depends on: AH-13

---

### M3 — Team Management (Backend)

**AH-17** Implement team member CRUD
`backend` · Priority: High · M3
- internal/team/model.go, repository.go, service.go, handler.go, routes.go
- GET /team (all users — super_admin only)
- POST /team/invite (send invite email via Resend)
- Depends on: AH-9, AH-8

**AH-18** Implement RBAC role management
`backend` `security` · Priority: High · M3
- POST /team/roles/assign (super_admin only — direct role grant)
- POST /team/roles/requests (admin+ — email-based promotion request)
- POST /team/roles/requests/:id/approve (super_admin only)
- POST /team/roles/requests/:id/reject (super_admin only)
- Audit log every role change
- Depends on: AH-17, AH-9

**AH-19** Implement mailer service
`backend` · Priority: Medium · M3
- internal/mailer/service.go
- SendInvite, SendRoleApproved, SendRoleRejected
- Resend HTTP API
- Depends on: AH-3

---

### M4 — Projects & Dashboard (Backend)

**AH-20** Implement project CRUD
`backend` · Priority: High · M4
- internal/project/ — all files
- CRUD endpoints, status update endpoint
- GET /clients/:id/projects (projects for a specific client)
- Depends on: AH-9, AH-8

**AH-21** Implement dashboard aggregate endpoint
`backend` · Priority: Medium · M5
- internal/dashboard/ — all files
- GET /dashboard — returns: active client count, projects by status,
  recent 5 clients, Claude usage summary this month
- Depends on: AH-13, AH-20

---

### M6 — Frontend Shell

**AH-22** Set up React + Vite + TypeScript project
`frontend` `chore` · Priority: Urgent · M6
- Init Vite project with React 19 + TypeScript
- Install: TanStack Query v5, React Router v7, Tailwind v4, shadcn/ui,
  React Hook Form, Zod, Framer Motion, Sonner, Lucide React, axios
- Set up tailwind.config.ts, tsconfig.json, components.json
- Create folder structure skeleton

**AH-23** Set up axios client + interceptors
`frontend` `security` · Priority: Urgent · M6
- src/lib/axios.ts
- CSRF fetch on app init, attach to mutations
- 401 → refresh → retry → redirect
- 429 → Sonner toast
- Depends on: AH-22

**AH-24** Set up auth context + TanStack Query provider
`frontend` · Priority: Urgent · M6
- src/contexts/auth-context.tsx
- TanStack Query QueryClientProvider
- Depends on: AH-22

**AH-25** Set up React Router + central route registry
`frontend` · Priority: Urgent · M6
- src/config/routes.tsx — all routes with lazy imports
- ProtectedRoute, GuestRoute, RoleProtectedRoute guards
- Depends on: AH-24

**AH-26** Design: Login page (Variants)
`design` · Priority: High · M6
- Create login page design in Variants
- Full-page centered card layout
- Agency branding, email + password fields, submit button

**AH-27** Build: Login page + form
`frontend` · Priority: High · M6
- src/features/auth/pages/LoginPage.tsx
- src/features/auth/components/LoginForm.tsx
- React Hook Form + Zod, error states, lockout message
- Adapted from Variants design (AH-26)
- Depends on: AH-25, AH-26

**AH-28** Design: App shell — sidebar + header (Variants)
`design` · Priority: High · M6
- Design AppSidebar and AppHeader in Variants
- Dark sidebar variant preferred for internal tool feel
- Nav sections: Dashboard, Clients, Projects, Team (role-aware visibility)
- User footer: avatar, name, role badge, logout button

**AH-29** Build: AppLayout + AppSidebar + AppHeader
`frontend` · Priority: High · M6
- src/components/layout/AppLayout.tsx
- src/components/layout/AppSidebar.tsx
- src/components/layout/AppHeader.tsx
- Role-aware link visibility, active route highlighting
- Adapted from Variants design (AH-28)
- Depends on: AH-24, AH-28

---

### M7 — Client Management UI

**AH-30** Design: Clients table + client card (Variants)
`design` · Priority: High · M7
- Clients list table with status badges, plan tier, actions column
- Client card for compact views

**AH-31** Build: Clients list page
`frontend` · Priority: High · M7
- src/features/clients/pages/ClientsPage.tsx
- src/features/clients/components/ClientsTable.tsx
- Sortable columns, search bar, status filter, "New Client" button
- Adapted from Variants design (AH-30)
- Depends on: AH-29, AH-13

**AH-32** Design: Client detail page (Variants)
`design` · Priority: High · M7
- Full client profile layout
- Token management section: management token card + connection token card
- Deploy status card, Supabase info card, project list

**AH-33** Build: Client detail page
`frontend` · Priority: High · M7
- src/features/clients/pages/ClientDetailPage.tsx
- All sub-components: ClientForm (edit), TokenManager, TokenRevealModal,
  DeployStatusCard, SupabaseInfoCard
- TokenRevealModal: copy-to-clipboard, "I've saved this" confirm button
- Adapted from Variants design (AH-32)
- Depends on: AH-31, AH-14, AH-15, AH-16

**AH-34** Build: New client form
`frontend` · Priority: High · M7
- src/features/clients/pages/NewClientPage.tsx
- Reuses ClientForm, handles POST /clients + auto-triggers token generation
- Shows TokenRevealModal immediately after creation
- Depends on: AH-33

---

### M8 — Projects UI

**AH-35** Design: Projects kanban board (Variants)
`design` · Priority: Medium · M8
- Status columns: discovery, design, development, review, live, maintenance
- Project cards with client name, due date, assigned to

**AH-36** Build: Projects kanban + detail
`frontend` · Priority: Medium · M8
- src/features/projects/pages/ProjectsPage.tsx (kanban view)
- src/features/projects/pages/ProjectDetailPage.tsx
- Status update via PATCH /projects/:id/status
- Adapted from Variants design (AH-35)
- Depends on: AH-29, AH-20

---

### M9 — Team UI

**AH-37** Design: Team management page (Variants)
`design` · Priority: Low · M9
- Table of team members with role badges
- Invite modal, role assignment modal

**AH-38** Build: Team management page
`frontend` · Priority: Low · M9
- src/features/team/pages/TeamPage.tsx
- TeamTable, InviteModal, RoleAssignModal
- Visible only to super_admin
- Adapted from Variants design (AH-37)
- Depends on: AH-29, AH-17, AH-18

---

### M10 — Dashboard UI

**AH-39** Design: Dashboard overview (Variants)
`design` · Priority: Medium · M10
- Stats row: 4 metric cards
- Recent clients list
- Projects status overview (mini board or chart)

**AH-40** Build: Dashboard page
`frontend` · Priority: Medium · M10
- src/features/dashboard/pages/DashboardPage.tsx
- StatsGrid, StatCard, RecentClients, ProjectsOverview
- Adapted from Variants design (AH-39)
- Depends on: AH-29, AH-21

---

### M11 — QA & Launch

**AH-41** End-to-end test: Auth flow
`backend` `frontend` · Priority: High · M11
- Login → dashboard → logout → redirect to login
- Expired token → refresh → resume session
- Account lockout after 5 failed logins

**AH-42** End-to-end test: Client lifecycle
`backend` `frontend` · Priority: Urgent · M11
- Create client → management_token generated + shown in modal
- Generate connection_token → shown once
- Revoke connection_token
- Update client record → verify in Supabase

**AH-43** End-to-end test: Token validation endpoint
`backend` · Priority: Urgent · M11
- Simulate portal backend calling GET /validate-management-token
- Valid token → 200 + client info
- Invalid token → 401
- Revoked client → 401

**AH-44** RBAC test: Role-based access enforcement
`backend` `frontend` · Priority: High · M11
- Moderator cannot POST to /clients (expects 403)
- Admin cannot POST to /team/roles/assign (expects 403)
- super_admin can do everything

**AH-45** Security review
`security` · Priority: High · M11
- Verify all security headers present
- Verify CSRF token required on all mutations
- Verify rate limits enforced on auth routes
- Verify token hashing (no plaintext in DB)
- Verify audit logs populated for all tracked actions

**AH-46** Deploy to Railway + Vercel
`infra` · Priority: Urgent · M11
- Backend: Railway deployment, all env vars set
- Frontend: Vercel deployment, VITE_API_BASE_URL set
- Verify /health endpoint reachable
- Verify Swagger UI accessible (internal only — add IP restriction if needed)

---

## Issue Dependency Graph (Summary)

```
AH-1,2,3 → AH-4 → AH-5,6
AH-6 → AH-7,8,9 → AH-10,11
AH-9 → AH-13 → AH-14 → AH-15
AH-13 → AH-16, AH-20
AH-9 → AH-17 → AH-18
AH-3 → AH-19

AH-22 → AH-23,24 → AH-25
AH-25 → AH-26,27 (auth UI)
AH-24 → AH-28,29 (shell)
AH-29 → AH-30,31 → AH-32,33 → AH-34 (clients UI)
AH-29 → AH-35,36 (projects UI)
AH-29 → AH-37,38 (team UI)
AH-29 → AH-39,40 (dashboard UI)

All → AH-41..46 (QA)
```

---

## Working Agreement

- All backend issues merged to `develop` before any dependent frontend issue is started
- Design issue (Variants) must be marked Done before its Build counterpart is started
- No issue moves to Done without a smoke test
- Security issues reviewed by at least one other team member before merge
