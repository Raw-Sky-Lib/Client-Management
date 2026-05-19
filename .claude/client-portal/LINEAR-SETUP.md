> **Historical reference.** All CLI issues (CLI-5 through CLI-45) are complete.
> See `Client-Management/.claude/todo.md` for the full completed checklist.

# client-portal — Linear Project (Completed)

```
Team:       client-portal
Identifier: CLI (issues: CLI-5 through CLI-45)
Status:     All issues completed as of current build
```

## What Was Built (by Cycle)

```
Cycle 1 (M1): Backend Foundation
  CLI-5  Go project setup
  CLI-6  Supabase project + migrations
  CLI-7  Startup management token validation
  CLI-8  Config, DB, middleware stack
  CLI-9  Onboarding flow (now replaced by magic-link — see FLOW.md)
  CLI-10 Portal auth (JWT with lean claims)
  CLI-11 Tenant registry service + credential encryption
  CLI-12 ISR revalidation service

Cycle 2 (M2): Claude Content Assistant (Backend)
  CLI-13 Rate limiter (Redis sliding window)
  CLI-14 Usage repository (agency-hub API calls)
  CLI-15 Prompt builder
  CLI-16 Claude service + handler

Cycle 3 (M3): Frontend Shell
  CLI-17 React + Vite + TypeScript setup
  CLI-18 Axios client + Supabase context (SupabaseProvider, useTenantSupabase)
  CLI-19 Auth context + routes (ProtectedRoute, GuestRoute, AuthOnlyRoute)
  CLI-20 Onboarding page design
  CLI-21 Onboarding flow UI (now /welcome + auth-callback)
  CLI-22 Portal shell design
  CLI-23 PortalLayout + sidebar + header

Cycle 4 (M4+M5): Content Editors
  CLI-24–27 Pages list + page editor + section editors
  CLI-28–30 Blog list + Tiptap post editor

Cycle 5 (M6+M7): Media + Claude UI
  CLI-31–33 Media library + MediaPickerModal
  CLI-34–36 Claude assistant UI

Cycle 6 (M8+M9): Secondary Features + QA
  CLI-37 Form submissions inbox
  CLI-38 Settings pages
  CLI-39 Dashboard overview
  CLI-40–45 E2E flows + security review + deploy
```
