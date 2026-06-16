# tasks.md — client-portal (finalization)
> The original CLI-5 → CLI-45 buildout in `todo.md` is complete.
> This file tracks the **finalization punch list** — bugs, UX rework, brand realignment, and the unfinished editor work currently sitting in the working tree.

---

## Where we are

**Shipped:** auth (password + magic link + reset), tenant registration via agency-hub push, multi-project switcher, pages + section editors, blog + Tiptap, media library (storage-native), Claude assistant, forms inbox, settings, dashboard, ISR revalidation.

**Currently broken or rough:**
1. **Login** — regressions surfaced after `1148b56 feat: Robust Error Handler` and `730fbe7 feat: Editing Feature Setup`. Specific failure mode needs reproduction (see CP-1).
2. **Page editor flow feels gimmicky** — `page-editor-page.tsx` gates the left-side field editor on a click *inside the cross-origin iframe* (`PORTAL_SECTION_CLICK` postMessage). When the handshake doesn't fire (origin mismatch, hot reload, client site not implementing the protocol), you can only "Jump to section" from a small list at the bottom. The whole editing experience hinges on a postMessage that the client template has to opt into.
3. **Brand styleguide drift** — portal still ships the cream / ink / forest / brand-red palette in `bg-cream`, `border-ink`, `bg-forest`, hard-shadow cards. Agency direction has moved; portal hasn't.
4. **Uncommitted in-flight work** — four new untracked files for `contact-editor`, `embed-editor`, `embed-code-field`, `image-picker-field`. They're referenced by `section-editor.tsx` already, so the build is fine, but they're not in git. The matching `ContactSection` / `EmbedSection` shapes are already in `types/index.ts`.
5. **Backend in-flight** — `api/internal/media/handler.go` (+113 lines), `media/routes.go` (+1), `onboarding/service.go` (-23 lines, trimmed). Need to audit what changed and whether it's complete.

---

## Plan

Four-phase finalization. Don't bundle phases — each lands as its own PR so we can roll back cleanly if something regresses.

**Phase 1 — Stabilize what we have** (CP-1 → CP-4)
Reproduce the login break, audit the in-flight backend diff, commit the new editor files cleanly. No new features. Goal: green build, clean working tree, login works end-to-end on a fresh tenant.

**Phase 2 — Visual editor rebuild via an editing bridge** (CP-5 → CP-11)
Replace the current "click section → edit on the left" flow with true inline editing inside the iframe — click a headline to type, click an image to swap, add/remove/reorder list items with floating toolbars. The template (`format-studio-client-site-template`) becomes an active partner by wrapping its fields in a tiny `<Editable>` / `<EditableImage>` / `<EditableList>` bridge. The portal stays the source of truth for state, dirty tracking, validation, and publish; the template only lends its DOM as the editing surface. Cross-origin safe — all communication via `postMessage`. The "Important features only" scope: inline text, image swap, list ops, section hide/show. SEO meta + raw JSON go in a slim on-demand drawer, not an always-on sidebar.

**Phase 3 — Brand realignment** (CP-12 → CP-14)
Confirm the new palette/type system, then sweep tokens in `index.css` + Tailwind config. Component-level classes (`bg-cream`, `bg-ink`, `bg-forest`, `bg-brand-red`, `font-sans`, `font-mono`, `border-ink`, hard-shadow utilities) should map through tokens, not get rewritten one-by-one — so a future brand change is a token swap, not a sweep.

**Phase 4 — Polish + ship** (CP-15 → CP-18)
Loose ends: error toasts for cross-origin iframe failures, empty-state copy on every CMS page, agency-hub ↔ portal end-to-end test on a real tenant, README pass.

---

## Legend
- `[ ]` not started · `[→]` in progress · `[✓]` done · `[⊘]` blocked
- 🔴 urgent · 🟠 high · 🟡 medium · ⚪ low

---

## Phase 1 — Stabilize

- [✓] **CP-1** 🔴 Reproduce + fix the login regression (2026-06-16 — CSRF bootstrap decoupled from profile fetch + cookie-fallback in axios; backend now distinguishes DNS failure from generic Supabase unreachable with code + dev-mode hint in the UI)
  - Walk the full path: `/login` (password) → `/api/auth/login` → cookie set → `useAuth().refresh()` → `/dashboard`.
  - Walk magic-link path: `/login` → `/api/auth/magic-link` → email → `/auth/callback#access_token=…` → `/api/auth/exchange` → `refresh()` → destination (`/welcome` for invite/signup, else `/dashboard`).
  - Suspect surface: the "Robust Error Handler" commit changed error shapes — check `axios.ts` interceptor + the `parseApiError` path. The 401-refresh-retry loop in particular.
  - Suspect surface: `onboarding/service.go` shrank by 23 lines — confirm the magic-link invite still wires through correctly on `POST /api/admin/register-client`.
  - Acceptance: clean tenant from agency-hub push → invite email arrives → magic link → `/welcome` → set password → next login (password) works → next login (magic) works → refresh on dashboard works → logout → login again works.

- [✓] **CP-2** 🟠 Audit + finish the in-flight backend diff (2026-06-16 — `CreateFolder` endpoint reviewed clean; `bucketNameFromSiteURL` consistent across media + onboarding; fixed `EMPTY_ITEM` `avatar` → `avatar_url` rename bug in testimonials editor; backend + frontend both compile)
  - Read `api/internal/media/handler.go` end-to-end and confirm the 113-line addition is intentional (likely new signed-URL / folder-create endpoints).
  - Confirm the new route in `media/routes.go` is registered with auth + CSRF middleware appropriately.
  - Confirm `onboarding/service.go` trim didn't remove a still-needed branch (the invite-send path is the high-risk one).
  - Commit as a single PR with tests for the new media endpoints.

- [✓] **CP-3** 🟠 Commit the four untracked editor files (2026-06-16 — audited clean, frontend `tsc --noEmit` passes; commit boundaries presented to user; also synced editing-bridge module from template into `client-dagim-digital-agency` so live tenant can test)
  - `contact-editor.tsx`, `embed-editor.tsx`, `embed-code-field.tsx`, `image-picker-field.tsx`.
  - Already wired into `section-editor.tsx` and types — just stage, smoke-test the contact + embed flow against a test tenant, commit.
  - Acceptance: a contact section saved from the portal renders correctly on the client template; an embed section's HTML survives the round-trip without re-escaping.

- [→] **CP-4** 🟡 Clean working tree + remove `api/tmp/main` from diff (2026-06-16 — in progress)
  - `api/tmp/main` should be gitignored (build artifact). Confirm `.gitignore` covers `api/tmp/` and untrack the binary.
  - `.claude/settings.json` diff — review what changed and decide if it belongs in the repo or in `settings.local.json`.

---

## Phase 2 — Visual editor rebuild via an editing bridge

> Mental model: the iframe IS the editor. The portal manages state and chrome.
> Sequencing matters here — protocol first, then template wiring, then portal rewrite. Don't start the portal rewrite until at least one template section is wired up end-to-end.

- [✓] **CP-5** 🔴 Write the bridge design doc — `.claude/EDITING-BRIDGE.md` (2026-06-16 — approved by user, defaults locked in §11)
  - Message protocol (every postMessage shape, both directions, with TypeScript types).
  - React component API: `<Editable>`, `<EditableImage>`, `<EditableList>`, `useEditMode()`.
  - Edit-mode activation: URL param + handshake token (so a random parent can't activate edit mode on a public site).
  - Dirty state model: portal is the only source of truth; iframe is presentation.
  - Lifecycle scenarios: load, text edit, image swap, list reorder, publish, discard, focus / undo / scroll-restore.
  - Security: origin validation, edit-mode token verification, what the bridge must NEVER expose.
  - Explicit out-of-scope list: no drag-to-add new sections, no layout/style/color editing, no animation editor. Content only.
  - Acceptance: a different engineer could implement either side (template or portal) from this doc alone.

- [→] **CP-6** 🟠 Build the `editing-bridge` module in the template repo (2026-06-16 — module + test harness built; integration testing parked until CP-1..CP-4 land. **Do not progress to CP-7 until CP-1..CP-4 are green.**)
  - Lives in `format-studio-client-site-template` as `src/lib/editing-bridge/` (or as a small standalone package if we later want to publish it).
  - Exports: `<Editable path="…">`, `<EditableImage path="…">`, `<EditableList path="…">`, `<EditableSection type="…">`, `useEditMode()`, `EditingBridgeProvider`.
  - In production (no edit mode active) every component renders as a transparent pass-through — zero runtime overhead, zero DOM footprint.
  - In edit mode: contenteditable for text, click-target + floating "Change" affordance for images, inline ⊕ / ✕ / drag-handle for list items, section hover outline + visibility tag.
  - Handshake: on mount, post `BRIDGE_READY { protocolVersion, sectionsRendered }`; wait for `PORTAL_ACTIVATE { token }` before activating.
  - Acceptance: bridge module imported but never activated = zero visible / behavioral change in production builds.

- [ ] **CP-7** 🟠 Wire the bridge into every template section
  - One PR per section: hero → about → features → testimonials → CTA → why_us → process → featured_projects → contact → embed.
  - Each PR wraps the field reads with `<Editable path="hero.headline">{headline}</Editable>` style — minimal refactor.
  - Validate against a staging tenant after each: live editing of that section's fields posts the correct `FIELD_CHANGE` shape; production render is unchanged.
  - Test on an exsisitng client repo called dagim-digital-agency it's level with this repo's folder.
  - Acceptance: every field in every `*Section` type in `web/src/types/index.ts` (portal repo) is reachable via at least one bridge wrapper.

- [ ] **CP-8** 🔴 Rewrite `page-editor-page.tsx` as a full-bleed visual editor
  - Iframe takes the full canvas. No sidebar in edit mode.
  - Top chrome: page title + status pill, section quick-nav dropdown (replaces the current sidebar list), preview/edit toggle, mobile/desktop toggle, refresh.
  - Bottom chrome: floating publish bar (sticky, only visible when dirty). Shows dirty-section count + Discard / Publish.
  - Portal sends `PORTAL_ACTIVATE { token }` after iframe `BRIDGE_READY`, receives `FIELD_CHANGE` / `LIST_OP` / `SECTION_TOGGLE` events, applies to `localSections`, posts `PORTAL_UPDATE_SECTION` so the iframe re-renders.
  - Preview mode unchanged — iframe alone, no activation, no bridge UI.

- [ ] **CP-9** 🟠 MediaPicker as iframe-aware overlay
  - When the iframe sends `REQUEST_MEDIA_PICKER { path, anchorRect }`, portal opens the existing `MediaPickerModal` positioned near the clicked image (use `anchorRect` to anchor a popover; fall back to centered modal on mobile).
  - On select: portal sends `SET_FIELD { path, value }` → iframe updates → iframe posts back `FIELD_CHANGE`.

- [ ] **CP-10** 🟠 Slim "details" drawer for advanced fields
  - Right-edge drawer, opens on demand from the top chrome ("Details" button).
  - Contains: page slug, SEO title, SEO description, Open Graph image, raw JSON inspector for any unknown fields the bridge can't reach.
  - This is the *only* surviving piece of the old sidebar — used rarely, hidden by default.

- [ ] **CP-11** 🟡 Retire the old section editors
  - `web/src/features/pages/components/{hero,about,features,testimonials,cta,why-us,process,featured-projects,contact,embed}-editor.tsx`, plus `section-editor.tsx`, `editor-primitives.tsx`, `image-picker-field.tsx`, `embed-code-field.tsx` — all become unreachable once CP-8 ships.
  - Keep them in a single archive commit (or git tag) for one release in case we need to revert, then delete.
  - Update `.claude/CLAUDE.md` "Frontend Feature Structure" to reflect the new layout.

---

## Phase 3 — Brand realignment

- [ ] **CP-12** 🔴 Confirm the new brand direction
  - Inputs needed: primary palette tokens (cream / ink / forest / red are the current ones — what replaces them?), type pairing (currently sans + mono), card / shadow language (currently 8px hard shadow + ink border).
  - Deliverable: a one-page style spec written into `.claude/STYLE-GUIDE.md`.
  - **Blocks CP-13, CP-14.**

- [ ] **CP-13** 🟠 Centralize tokens in `index.css` + Tailwind theme
  - Move `cream`, `ink`, `forest`, `brand-red` into CSS variables under `:root` (or `@theme inline` for Tailwind v4).
  - Replace any direct hex codes in component files with the token classes.
  - Introduce semantic aliases: `bg-surface`, `bg-foreground`, `text-muted`, `border-default` — so component code uses intent, not raw color names.

- [ ] **CP-14** 🟠 Sweep components against the new spec
  - Login, auth-callback, welcome, portal-layout (sidebar + header), dashboard cards, page editor chrome, blog editor, settings tabs.
  - Use `HardShadowCard` and `AgencyBadge` as the canonical primitives — if they don't match the new spec, fix them once at the source, not at every call-site.

---

## Phase 4 — Polish + ship

- [ ] **CP-15** 🟡 Empty-state pass on every CMS page
  - Pages list, blog list, forms inbox, media library, settings — every one should have a clear "Nothing here yet" state with the next action.

- [ ] **CP-16** 🟡 Cross-origin iframe failure surfacing
  - If `activeProject.site_url` is unreachable, show a friendlier inline message in the editor preview area than the current `Monitor` placeholder. Include a "Test connection" button that hits the site URL and surfaces the response code.
  - Also surface a clear state if the iframe loads but never posts `BRIDGE_READY` within 3 s — likely means the tenant site is on an older template build.

- [ ] **CP-17** 🟡 End-to-end happy-path test on a real tenant
  - Agency-hub: create client → save Supabase credentials → push to portal.
  - Portal: receive invite → set password → edit a page inline → publish → confirm ISR fired → confirm content live on the client site.
  - Log every step, write a short runbook in `.claude/FLOW.md` (already exists — update it).

- [ ] **CP-18** ⚪ README + onboarding-doc pass
  - Update `web/README.md` and `api/.env.example` to reflect any new env vars introduced during CP-1 → CP-17.

---

## Open questions (need user input)

- **CP-12** Brand palette / type direction — current vs. new spec.
- **CP-1** Exact symptom of the login break — is it the password path, the magic-link path, both? Any console / network error to attach?
- **CP-7 sequencing** — wire all sections in one PR or one per section? Recommend one per section; ten small PRs is easier to review and revert.
