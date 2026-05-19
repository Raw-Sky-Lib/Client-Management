> **Canonical source:** See `Client-Management/.claude/CLAUDE.md`

# client-portal — Project Structure (Summary)

## Backend Internal Packages

```
internal/
├── onboarding/    RegisterClient (from agency-hub), SendInvite, ResendInvite,
│                  Confirm, DeregisterClient
├── auth/          Login (password), MagicLink, Exchange (Supabase token → portal JWT),
│                  Refresh, Logout, Profile, SetPassword,
│                  ResetPasswordRequest/Verify/Confirm, CSRF
├── portalproject/ GET /api/projects → returns decrypted project list
├── tenant/        ResolveTenant middleware — decrypt credentials from tenant_projects
│                  into request context
├── claude/        Rate limit (Redis), budget check (agency-hub API),
│                  Anthropic call, usage recording
├── revalidate/    POST /api/revalidate → fire-and-forget ISR to client site
├── media/         Signed URL generation for Supabase Storage
├── startup/       ValidateManagementToken on boot (os.Exit(1) if fails)
├── mailer/        Pluggable: Resend or Brevo (MAILER_PROVIDER env)
├── middleware/    JWT auth, CSRF, rate limit, CORS, security headers, logger
├── config/        LoadConfig()
├── database/      pgxpool + MigrateClientDB (runs CMS migrations on client Supabase)
└── utils/         crypto.go, errors.go, response.go, supabase.go
```

## Portal DB Tables

```
tenants(id, onboarded_at, created_at)
tenant_projects(id, tenant_id, agency_project_id, name,
                supabase_url_encrypted, supabase_anon_encrypted,
                supabase_service_role_encrypted, supabase_db_url_encrypted,
                site_url, created_at)
tenant_users(id, tenant_id, email, created_at)
email_confirmations(id, tenant_id, project_id, email, token_hash,
                    expires_at, used_at, created_at)
```

## Frontend Structure

```
src/
├── components/guards/    AuthOnlyRoute, GuestRoute, ProtectedRoute
├── contexts/
│   ├── auth-context.tsx        portal JWT, login/logout, useAuth()
│   └── supabase-context.tsx    Supabase client per project, useTenantSupabase()
├── features/
│   ├── onboarding/    welcome-page.tsx, link-error-page.tsx
│   ├── auth/          login-page.tsx, auth-callback-page.tsx, reset-password-page.tsx
│   ├── dashboard/     dashboard-page.tsx + widgets
│   ├── pages/         pages-list-page.tsx, page-editor-page.tsx + section editors
│   ├── blog/          blog-list-page.tsx, new/edit-post-page.tsx + tiptap-editor
│   ├── media/         media-page.tsx + file-browser, storage-item, media-picker-modal
│   ├── forms/         forms-page.tsx + submissions-table, submission-detail
│   ├── settings/      settings-page.tsx (general, SEO, social, nav editor)
│   └── assistant/     assistant-page.tsx + instruction-form, diff-preview, apply-bar
└── lib/
    ├── axios.ts        portal backend calls (CSRF, 401 refresh, 429 toast)
    └── utils.ts        cn(), formatDate(), formatBytes(), slugify()
```
