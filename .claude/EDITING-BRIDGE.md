# EDITING-BRIDGE.md
> Design spec for the inline visual content editor.
> Cross-origin iframe + `postMessage`. Portal = state. Template = editing surface.
> v1 — content only. No layout, no styles, no animations.

---

## 1. Overview

The portal renders the client's live site inside an iframe. The site template ships a small **editing-bridge** module that, when activated, makes individual content fields editable in place — click a headline to type, click an image to swap, hover a testimonial to delete it. Every change is posted back to the portal as a structured event. The portal owns dirty state, undo, validation, and publish. The template just lends its DOM.

```
┌──────────── Portal (portal.formatstudio.com) ────────────┐
│                                                          │
│   page-editor-page.tsx                                   │
│   ├─ localSections: PageSection[]   (source of truth)    │
│   ├─ iframe ──────────────────────────────────────────┐  │
│   │                                                   │  │
│   │   Template site (acme.com)                        │  │
│   │   ├─ <EditingBridgeProvider>                      │  │
│   │   ├─ <HeroSection>                                │  │
│   │   │    <Editable path="hero.headline">…</Editable>│  │
│   │   │    <EditableImage path="hero.image_url" />    │  │
│   │   ├─ <TestimonialsSection>                        │  │
│   │   │    <EditableList path="testimonials.items">…  │  │
│   │   └─ …                                            │  │
│   │                                                   │  │
│   └───────────────────────────────────────────────────┘  │
│                                                          │
│   ↑ FIELD_CHANGE / LIST_OP / REQUEST_MEDIA_PICKER         │
│   ↓ PORTAL_ACTIVATE / PORTAL_UPDATE_SECTIONS / …          │
└──────────────────────────────────────────────────────────┘
```

---

## 2. Activation flow

The bridge is **inert by default**. It only activates when all three are true:

1. URL has `?portal=edit` (the trigger)
2. Window is embedded — `window.top !== window`
3. Parent origin matches the template's compiled-in `EDITING_BRIDGE_ALLOWED_ORIGIN`

Sequence:

```
Portal                                     Template iframe
  │                                                  │
  │  src = "https://acme.com/?portal=edit"           │
  │ ───────────────────────────────────────────────► │
  │                                                  │  bridge mounts inert,
  │                                                  │  detects ?portal=edit,
  │                                                  │  detects embedded
  │                                                  │
  │ ◄──────────────────  BRIDGE_READY ───────────────│  postMessage('*', …)
  │  (event.origin = "https://acme.com")             │
  │                                                  │
  │  origin check ✓                                  │
  │  portal stores acme's origin as iframeOrigin     │
  │                                                  │
  │ ─────────────────  PORTAL_ACTIVATE ─────────────►│  postMessage(acme, …)
  │                                                  │
  │                                                  │  bridge stores
  │                                                  │  event.origin as
  │                                                  │  portalOrigin,
  │                                                  │  activates editing UI
  │                                                  │
  │ ─────────────  PORTAL_UPDATE_SECTIONS ─────────► │  (initial content)
  │                                                  │
```

**Standalone visit to `?portal=edit`** — `window.top === window` → bridge stays inert. No editing UI appears.

**Malicious parent embeds site + spoofs PORTAL_ACTIVATE** — origin check rejects. Bridge stays inert.

**Production visit** (no query param) — bridge module is imported but the `<Editable>` wrappers all render as transparent pass-throughs. Zero DOM footprint. Zero runtime overhead beyond a single `if (!active) return children;` per wrapper.

---

## 3. Message protocol

All messages are JSON-serializable objects with a `type` discriminator and a `protocolVersion: 1`. The bridge ignores any message whose `type` it doesn't recognize. Both sides MUST validate `event.origin` against their known counterpart before handling.

### 3.1 Iframe → Portal

```typescript
type BridgeOutbound =
  | { type: 'BRIDGE_READY';        protocolVersion: 1; manifest: Manifest }
  | { type: 'FIELD_CHANGE';        protocolVersion: 1; path: string; value: string }
  | { type: 'REQUEST_MEDIA_PICKER'; protocolVersion: 1; path: string; currentUrl: string | null; anchorRect: Rect }
  | { type: 'LIST_OP';             protocolVersion: 1; path: string; op: ListOp }
  | { type: 'SECTION_FOCUS';       protocolVersion: 1; sectionType: string }
  | { type: 'SECTION_TOGGLE';      protocolVersion: 1; sectionType: string; visible: boolean }

type ListOp =
  | { kind: 'add';     index: number }              // insert blank at index
  | { kind: 'remove';  index: number }
  | { kind: 'reorder'; from: number; to: number }

type Rect = { top: number; left: number; width: number; height: number }  // iframe-relative

type Manifest = {
  sectionsRendered: string[]   // e.g. ['hero','features','testimonials','cta']
  editablePaths:    string[]   // e.g. ['hero.headline','hero.cta_label','testimonials.items[]']
}
```

- `BRIDGE_READY` — sent once per page load, after the React tree mounts. The `manifest` lets the portal warn when a path it expects (per the section type) isn't reachable in the rendered DOM.
- `FIELD_CHANGE` — fired on **blur** of a contenteditable field, not on keystroke. Avoids re-render churn and focus loss.
- `REQUEST_MEDIA_PICKER` — `anchorRect` is in iframe-viewport coordinates; portal converts to its own viewport for the popover anchor.
- `LIST_OP` — for `add`, portal inserts a blank item using the section's default-item shape (defined in the portal's `types/index.ts` companion). For `reorder`, `from` and `to` are pre-move indices.
- `SECTION_FOCUS` — fired when user clicks anywhere inside a section. Portal uses this to highlight current section in the quick-nav.
- `SECTION_TOGGLE` — only used if the template renders a visibility chip on hover. Optional in v1.

### 3.2 Portal → Iframe

```typescript
type PortalOutbound =
  | { type: 'PORTAL_ACTIVATE';        protocolVersion: 1 }
  | { type: 'PORTAL_DEACTIVATE';      protocolVersion: 1 }
  | { type: 'PORTAL_UPDATE_SECTIONS'; protocolVersion: 1; sections: PageSection[] }
  | { type: 'PORTAL_SCROLL_TO';       protocolVersion: 1; sectionType: string }
  | { type: 'PORTAL_FOCUS_FIELD';     protocolVersion: 1; path: string }
```

- `PORTAL_ACTIVATE` / `PORTAL_DEACTIVATE` — toggle the editing UI.
- `PORTAL_UPDATE_SECTIONS` — the canonical state sync. Sent after **every** mutation the portal applies, so the iframe always renders the portal's authoritative state. Replaces existing `PORTAL_UPDATE_SECTION` (singular).
- `PORTAL_SCROLL_TO` — bridge scrolls the section into view smoothly.
- `PORTAL_FOCUS_FIELD` — bridge moves caret into the named field (used after `LIST_OP add` so the newly-inserted item gets focus immediately).

---

## 4. React component API

All components live in `format-studio-client-site-template/src/lib/editing-bridge/`.

### 4.1 `<EditingBridgeProvider>`

Mounts the bridge runtime once at the app root.

```tsx
<EditingBridgeProvider allowedPortalOrigin={import.meta.env.VITE_PORTAL_ORIGIN}>
  <App />
</EditingBridgeProvider>
```

Responsibilities: detect activation, install the `window.addEventListener('message')` listener, validate origin, hold `active` + `sections` in context, render the floating chrome (toolbars, hover outlines) into a portal-rooted overlay layer.

### 4.2 `<Editable path="hero.headline">`

Wraps a single string field.

```tsx
<h1 className="hero-headline">
  <Editable path="hero.headline">{headline}</Editable>
</h1>
```

- **Inert mode:** renders `{children}` directly, no wrapper element.
- **Active mode:** wraps in `<span contentEditable suppressContentEditableWarning onBlur={…} />`. On blur, posts `FIELD_CHANGE { path, value }`. Listens to incoming `PORTAL_UPDATE_SECTIONS` and updates innerText — but **only if the field is not currently focused** (prevents caret jump mid-typing).

Variant for URLs: `<Editable path="hero.cta_url" kind="url">{ctaUrl}</Editable>` — opens a small popover input on click instead of contenteditable.

### 4.3 `<EditableImage path="hero.image_url" />`

```tsx
<EditableImage
  path="hero.image_url"
  src={imageUrl}
  alt="hero"
  className="hero-image"
/>
```

- **Inert mode:** renders a plain `<img>`.
- **Active mode:** wraps image in a relative container. On hover, shows a translucent "Change image" pill. On click, posts `REQUEST_MEDIA_PICKER { path, currentUrl: src, anchorRect: getBoundingClientRect() }`.

### 4.4 `<EditableList path="testimonials.items">`

```tsx
<EditableList path="testimonials.items">
  {items.map((t, i) => (
    <TestimonialCard key={i} index={i} {...t}>
      <Editable path={`testimonials.items[${i}].quote`}>{t.quote}</Editable>
      <Editable path={`testimonials.items[${i}].author`}>{t.author}</Editable>
    </TestimonialCard>
  ))}
</EditableList>
```

- **Inert mode:** renders `{children}` directly.
- **Active mode:** on item hover, renders a floating toolbar near the top-right of that item with ⊕ above, ⊕ below, ✕ delete, ⠿ drag handle. The list itself becomes a `dnd-kit` sortable.
- The bridge needs to know "which child corresponds to which index" — children are addressed by their `key` (must equal the index or stable id). If keys mismatch, bridge falls back to DOM-order indexing.

### 4.5 `useEditMode()`

```tsx
const { active, focusedSection } = useEditMode()
```

For template components that want to render differently in edit mode (e.g. always-show carousel arrows instead of swipe-only on mobile).

---

## 5. Path grammar

Paths address a single value inside the `PageSection[]` array for the current page.

```
hero.headline                    → sections[type=hero].headline
hero.image_url                   → sections[type=hero].image_url
testimonials.items               → sections[type=testimonials].items  (whole array)
testimonials.items[2].quote      → sections[type=testimonials].items[2].quote
testimonials.items[2].avatar_url → image swap inside a list item
```

Rules:
- Section segment (`hero`, `testimonials`) is the section's `type` discriminator. Each type appears at most once per page.
- Numeric brackets `[N]` index arrays.
- Field segments are dot-separated `snake_case` matching the type interfaces in `web/src/types/index.ts`.

Portal-side path resolver lives in `web/src/features/pages/lib/path.ts` — both `getByPath(sections, path)` and `setByPath(sections, path, value)` (returns a new array, never mutates).

---

## 6. Dirty state + publish model

- Portal holds `localSections: PageSection[]` plus the persisted `page.sections`.
- `isDirty = !deepEqual(localSections, page.sections)` — recomputed in `useMemo`.
- The iframe is never authoritative. It only emits user intents (`FIELD_CHANGE`, `LIST_OP`) and renders whatever the portal sends via `PORTAL_UPDATE_SECTIONS`.
- On Publish: `PUT /api/cms/pages/{slug}/sections` → on success, invalidate query cache + fire ISR revalidate.
- On Discard: `setLocalSections(page.sections)` → portal sends `PORTAL_UPDATE_SECTIONS` → iframe reverts.
- Undo / redo (v1.1): portal keeps a ring buffer of `localSections` snapshots, each `FIELD_CHANGE`/`LIST_OP` pushes one. Out of scope for v1 launch.

---

## 7. Lifecycle scenarios

### 7.1 Edit a headline
1. User clicks headline in iframe.
2. Bridge: `<Editable>` wrapper sets `document.activeElement` to its contenteditable span.
3. User types. No messages sent yet.
4. User clicks elsewhere → blur fires.
5. Bridge posts `FIELD_CHANGE { path: 'hero.headline', value: '…' }`.
6. Portal mutates `localSections` via `setByPath`.
7. Portal posts `PORTAL_UPDATE_SECTIONS { sections: localSections }`.
8. Bridge receives it. The headline field is no longer focused → safe to update innerText.

### 7.2 Swap a hero image
1. User clicks hero image.
2. Bridge posts `REQUEST_MEDIA_PICKER { path: 'hero.image_url', currentUrl, anchorRect }`.
3. Portal opens `MediaPickerModal`, anchored near the iframe-relative `anchorRect` (translated to portal viewport coords).
4. User selects new image.
5. Portal mutates `localSections` with the new URL.
6. Portal posts `PORTAL_UPDATE_SECTIONS`. Modal closes.
7. Iframe re-renders with the new image.

### 7.3 Add a testimonial
1. User hovers existing item → bridge shows ⊕ button below.
2. User clicks ⊕.
3. Bridge posts `LIST_OP { path: 'testimonials.items', op: { kind: 'add', index: 3 } }`.
4. Portal builds a blank `TestimonialItem` from its known defaults and inserts at index 3.
5. Portal posts `PORTAL_UPDATE_SECTIONS`, then `PORTAL_FOCUS_FIELD { path: 'testimonials.items[3].quote' }`.
6. Bridge re-renders the list, scrolls new item into view, focuses the quote field.

### 7.4 Reorder testimonials
1. User drags item via the ⠿ handle. dnd-kit handles the visual drag locally.
2. On drop, bridge posts `LIST_OP { path: 'testimonials.items', op: { kind: 'reorder', from: 2, to: 0 } }`.
3. Portal applies the move and posts `PORTAL_UPDATE_SECTIONS`.

### 7.5 Leave edit mode
1. User clicks "Preview" in the portal chrome.
2. Portal posts `PORTAL_DEACTIVATE`. Bridge tears down editing UI; pages render as production.
3. iframe stays loaded — no reload. Switching back to "Edit" posts `PORTAL_ACTIVATE` again.

---

## 8. Security

- **Origin validation, both sides, every message.**
  - Bridge: `event.origin === allowedPortalOrigin` (compiled-in).
  - Portal: `event.origin === iframeOrigin` (derived from `new URL(activeProject.site_url).origin` and locked in after the first `BRIDGE_READY`).
- **Bridge stays inert if not embedded.** `window.top === window` → no UI, no event listener.
- **Bridge never executes content as HTML.** Text values are set via `innerText`, never `innerHTML`. The `embed` section is the one exception and goes through the slim "details" drawer with an explicit warning.
- **No secrets cross the bridge.** No JWT, no Supabase keys, no admin tokens. The portal already holds those.
- **Bridge cannot trigger writes to Supabase.** Only the portal writes, only on explicit Publish. The bridge is a pure UI proxy.

---

## 9. Edge cases

| Case | Handling |
|---|---|
| Field focused when `PORTAL_UPDATE_SECTIONS` arrives | Skip that field's innerText update until blur. |
| Iframe loaded but no `BRIDGE_READY` within 3 s | Portal shows a banner: "This site is on an older template — inline editing unavailable. Use the details drawer." Falls back to a minimal sidebar with raw field inputs. |
| Tenant site at `activeProject.site_url` is unreachable | Iframe load error → portal shows the same offline state as today, but with retry. |
| User opens portal in two tabs editing same page | Last-write-wins on Publish. v1 ships without conflict detection; document as known limitation. |
| Iframe scroll position on update | Bridge restores scrollTop after every `PORTAL_UPDATE_SECTIONS`. |
| Hot reload during template development | Bridge re-mounts → sends `BRIDGE_READY` again → portal re-activates. Idempotent. |
| Section appears in JSONB but template doesn't render it | `manifest.sectionsRendered` won't include it. Portal warns: "Section 'x' has saved content but isn't rendered on this site." |
| Unknown field in section JSONB | Bridge ignores; portal still exposes via the details drawer JSON inspector. |

---

## 10. Out of scope (v1)

- Drag-to-add new sections to a page. (Sections list is fixed per page; managed via Supabase migration when needed.)
- Layout / spacing / column controls.
- Styling — color, font, size. All driven by the template, not the editor.
- Animation / transition editing.
- Component-level reusable blocks (no symbols / instances).
- Multi-page navigation editing from inside the editor. (Nav lives in Settings, not in this surface.)
- Real-time collaboration / presence indicators.
- Undo / redo. (Tracked as a v1.1 fast-follow.)

---

## 11. Locked decisions

These were open during the design pass and are now settled. Recorded here so we don't relitigate during implementation.

- **Per-list defaults shape** — A static `DEFAULTS` map lives at `web/src/features/pages/lib/defaults.ts`, keyed by section type, returning the blank item shape for that section's list field. `LIST_OP add` reads from it. Editing the map is a one-line change when a new section type ships. No generated/runtime introspection.
- **`EDITABLE_PATHS` registry** — Trust the manifest. The bridge's `BRIDGE_READY.manifest.editablePaths` is the source of truth for which fields are reachable on the current site build. The portal compares against the expected set for each section type (derived from `web/src/types/index.ts`) and surfaces a non-blocking warning when they diverge — e.g. "field `hero.subheadline` is in your content but not rendered in this template build." Never blocks editing of the paths that ARE present.
- **Drag library** — `@dnd-kit/core` + `@dnd-kit/sortable`. Lives in the template's `editing-bridge` module, lazy-imported only when the bridge activates so production bundles stay clean. Accessibility (keyboard reorder) ships in v1.
- **Embed sections** — Details-drawer only. Inline editing of arbitrary HTML inside the iframe is a footgun (XSS surface, broken styling preview, hard to escape). The `<EditableSection type="embed">` wrapper renders the embed normally in edit mode but clicking it just posts `SECTION_FOCUS` — actual edits happen in the right-side details drawer with an explicit "this content is rendered as HTML" warning.

---

## 12. Done means

- Both repos can implement their side from this doc alone — no further design needed.
- Message shapes are stable; bumping `protocolVersion: 1 → 2` is the only way to evolve them.
- A new section type can be added with: (a) interface in `types/index.ts`, (b) `<*Section>` component in template wrapping fields with `<Editable>`, (c) one entry in `DEFAULTS` if it has lists. No portal code changes required.
