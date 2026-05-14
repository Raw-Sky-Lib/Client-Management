# Client Portal Setup Guide
> For AI assistants working on client site projects that connect to the agency's client-portal CMS.
> This file covers what needs to exist in the client site codebase and Supabase for the portal to work.

---

## What the Portal Manages

The client-portal CMS manages these content types in the client's Supabase project:

| Portal feature | Supabase table | Notes |
|----------------|----------------|-------|
| Settings → General/SEO/Social | `site_settings` | key/value rows, NOT a flat single-row table |
| Settings → Nav | `nav_items` | drag-to-reorder |
| Pages editor | `pages` | JSONB sections per page |
| Blog editor | `posts` | Tiptap HTML content |
| Media library | `media` + Supabase Storage | bucket named from site URL hostname |
| Form submissions inbox | `form_submissions` | `data` column is JSONB |

---

## Required Supabase Schema

All tables must match exactly. The portal queries specific column names — a schema mismatch causes silent failures.

```sql
-- Key/value store. Every setting is a separate row.
-- site_name, tagline, logo_url, contact_email, seo_title, seo_description,
-- og_image_url, social_links (JSON string) are all individual rows.
site_settings(
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key        TEXT NOT NULL UNIQUE,
  value      TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)

pages(
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug            TEXT NOT NULL UNIQUE,
  title           TEXT NOT NULL,
  sections        JSONB NOT NULL DEFAULT '[]',
  seo_title       TEXT,
  seo_description TEXT,
  is_published    BOOLEAN NOT NULL DEFAULT false,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
)

posts(
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug            TEXT NOT NULL UNIQUE,
  title           TEXT NOT NULL,
  content         TEXT,
  excerpt         TEXT,
  cover_image_url TEXT,
  author_name     TEXT,
  is_published    BOOLEAN NOT NULL DEFAULT false,
  published_at    TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
)

nav_items(
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  label       TEXT NOT NULL,
  url         TEXT NOT NULL,
  "order"     INTEGER NOT NULL DEFAULT 0,
  is_external BOOLEAN NOT NULL DEFAULT false
)

-- data is JSONB — all form fields stored as { name: "...", email: "...", message: "..." }
-- is_read is set to true by the portal when the submission is opened
form_submissions(
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  form_name    TEXT NOT NULL,
  data         JSONB NOT NULL DEFAULT '{}',
  is_read      BOOLEAN NOT NULL DEFAULT false,
  submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)

media(
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  filename    TEXT NOT NULL,
  url         TEXT NOT NULL,
  mime_type   TEXT NOT NULL,
  size_bytes  BIGINT NOT NULL,
  uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)
```

### RLS Policies Required

```sql
-- All CMS tables need public read
CREATE POLICY anon_read_site_settings ON site_settings FOR SELECT TO anon USING (true);
CREATE POLICY anon_read_pages ON pages FOR SELECT TO anon USING (true);
CREATE POLICY anon_read_posts ON posts FOR SELECT TO anon USING (true);
CREATE POLICY anon_read_nav_items ON nav_items FOR SELECT TO anon USING (true);
CREATE POLICY anon_read_media ON media FOR SELECT TO anon USING (true);

-- form_submissions: anon insert (contact form) + select/update (portal marks read)
CREATE POLICY anon_insert_form_submissions ON form_submissions FOR INSERT TO anon WITH CHECK (true);
CREATE POLICY anon_select_form_submissions ON form_submissions FOR SELECT TO anon USING (true);
CREATE POLICY anon_update_form_submissions ON form_submissions FOR UPDATE TO anon USING (true);
```

---

## Required: `/api/revalidate` Route

After every content save, the portal backend posts to this endpoint to trigger ISR. Every client site must implement it.

```typescript
// src/app/api/revalidate/route.ts  (Next.js App Router)
import { revalidatePath } from 'next/cache'
import { NextResponse } from 'next/server'

export async function POST(request: Request) {
  const secret = request.headers.get('x-revalidate-secret')
  if (secret !== process.env.REVALIDATE_SECRET) {
    return NextResponse.json({ error: 'invalid secret' }, { status: 401 })
  }
  const { paths } = await request.json() as { paths: string[] }
  for (const path of paths ?? []) {
    revalidatePath(path)
  }
  return NextResponse.json({ revalidated: true, paths })
}
```

Environment variable needed: `REVALIDATE_SECRET` — set in Vercel, value provided by the agency.

If the project currently uses time-based ISR (`export const revalidate = 3600`), keep it — it acts as a fallback.

---

## Required: Contact Form → Portal Inbox Format

If the client site has a contact form, it must write to `form_submissions` in this format for submissions to appear in the portal inbox:

```typescript
await supabase.from('form_submissions').insert({
  form_name: 'contact',             // identifies which form
  data: { name, email, message },   // any fields as JSONB — all shown in portal
})
```

Any fields inside `data` are displayed. Do not use flat columns — the portal reads only `form_name`, `data`, `is_read`, `submitted_at`.

---

## Schema Conflicts on Already-Started Projects

The portal's `register-client` endpoint runs migrations with `CREATE TABLE IF NOT EXISTS`. If a table already exists with a different schema, the migration silently skips it — but portal queries will then fail at runtime.

**Check for these conflicts before the client is registered:**

### `site_settings` as a flat single-row table

**Problem:** Many client sites use `site_settings` as one row with named columns (`site_name TEXT, tagline TEXT, ...`). The portal expects one row per setting with `key` and `value` columns.

**Fix:** Rename the existing table and create a fresh one.

In Supabase SQL editor:
```sql
ALTER TABLE site_settings RENAME TO site_config;
```

In the client site code, update all queries:
```typescript
// before
supabase.from('site_settings').select('*').single()

// after
supabase.from('site_config').select('*').single()
```

After this rename, the portal migration creates a correct `site_settings` key/value table. The client site keeps reading from `site_config` for its own flat data. Both coexist.

### `form_submissions` with flat columns

**Problem:** Some projects have `form_submissions(id, name, email, message, submitted_at)`. The portal expects `form_name TEXT, data JSONB, is_read BOOLEAN`.

**Fix:** Rename the existing table, update any queries that read from it, and repoint the contact form insert to use the portal format.

In Supabase SQL editor:
```sql
ALTER TABLE form_submissions RENAME TO contact_submissions;
```

Update the contact form insert:
```typescript
// before — flat columns
supabase.from('form_submissions').insert({ name, email, message })

// after — portal format (shows in portal inbox)
supabase.from('form_submissions').insert({
  form_name: 'contact',
  data: { name, email, message },
})
```

### `nav_items` — no conflict

The portal's `nav_items` schema (`id, label, url, order, is_external`) matches what most projects use. If the existing table has this schema, the migration is a safe no-op.

---

## Idempotent Migration Script

Run this in Supabase SQL editor on any new client project. Safe to run even if some tables already exist.

```sql
CREATE TABLE IF NOT EXISTS site_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT NOT NULL UNIQUE,
    value TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    sections JSONB NOT NULL DEFAULT '[]',
    seo_title TEXT,
    seo_description TEXT,
    is_published BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS pages_slug_idx ON pages(slug);

CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    content TEXT,
    excerpt TEXT,
    cover_image_url TEXT,
    author_name TEXT,
    is_published BOOLEAN NOT NULL DEFAULT false,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS posts_slug_idx ON posts(slug);

CREATE TABLE IF NOT EXISTS nav_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    label TEXT NOT NULL,
    url TEXT NOT NULL,
    "order" INTEGER NOT NULL DEFAULT 0,
    is_external BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS form_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    form_name TEXT NOT NULL,
    data JSONB NOT NULL DEFAULT '{}',
    is_read BOOLEAN NOT NULL DEFAULT false,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS form_submissions_submitted_at_idx ON form_submissions(submitted_at DESC);

CREATE TABLE IF NOT EXISTS media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename TEXT NOT NULL,
    url TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE site_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE pages ENABLE ROW LEVEL SECURITY;
ALTER TABLE posts ENABLE ROW LEVEL SECURITY;
ALTER TABLE nav_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE form_submissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE media ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename='site_settings' AND policyname='anon_read_site_settings') THEN
    CREATE POLICY anon_read_site_settings ON site_settings FOR SELECT TO anon USING (true); END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename='pages' AND policyname='anon_read_pages') THEN
    CREATE POLICY anon_read_pages ON pages FOR SELECT TO anon USING (true); END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename='posts' AND policyname='anon_read_posts') THEN
    CREATE POLICY anon_read_posts ON posts FOR SELECT TO anon USING (true); END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename='nav_items' AND policyname='anon_read_nav_items') THEN
    CREATE POLICY anon_read_nav_items ON nav_items FOR SELECT TO anon USING (true); END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename='media' AND policyname='anon_read_media') THEN
    CREATE POLICY anon_read_media ON media FOR SELECT TO anon USING (true); END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename='form_submissions' AND policyname='anon_insert_form_submissions') THEN
    CREATE POLICY anon_insert_form_submissions ON form_submissions FOR INSERT TO anon WITH CHECK (true); END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename='form_submissions' AND policyname='anon_select_form_submissions') THEN
    CREATE POLICY anon_select_form_submissions ON form_submissions FOR SELECT TO anon USING (true); END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename='form_submissions' AND policyname='anon_update_form_submissions') THEN
    CREATE POLICY anon_update_form_submissions ON form_submissions FOR UPDATE TO anon USING (true); END IF;
END $$;
```

---

## Matt Banton (`matt-banton-portfolio`) — What Needs to Change

This project was built in Phase 1 without portal wiring. The following code changes are needed before it can connect to the portal.

### Schema conflicts to resolve first (in Supabase SQL editor)

```sql
ALTER TABLE site_settings RENAME TO site_config;
ALTER TABLE form_submissions RENAME TO contact_submissions;
```

### Code changes

**`src/lib/queries.ts`** — update `getSiteSettings`:
```typescript
// change 'site_settings' → 'site_config'
const { data } = await supabase.from('site_config').select('*').limit(1).single()
```

**Contact form handler** (wherever it inserts to `form_submissions`) — switch to portal format:
```typescript
await supabase.from('form_submissions').insert({
  form_name: 'contact',
  data: { name, email, subject, message },
})
```

**`src/app/api/revalidate/route.ts`** — replace the Phase 1 stub (which just returns 200) with the real implementation shown above.

**Add to Vercel env vars:**
```
REVALIDATE_SECRET=<provided by agency>
```

### What the portal will manage after connection

- `site_settings` (key/value): site_name, tagline, logo_url, contact_email, SEO, social links
- `nav_items`: his existing nav rows (same schema — portal takes over management)
- `pages`: any new pages created via the portal editor
- `posts`: any blog content if added
- `media`: images uploaded via the portal (Storage bucket `mattbanton-com`)
- `form_submissions`: contact form submissions visible in portal inbox

### What the portal does NOT touch

His custom tables — `site_config`, `projects`, `project_images`, `gallery_images`, `gallery_collections`, `about_content`, `contact_submissions` — are invisible to the portal and remain unchanged.
