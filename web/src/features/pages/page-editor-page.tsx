import { useState, useEffect, useRef, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router'
import { useQueryClient } from '@tanstack/react-query'
import {
  ChevronLeft, AlertCircle, ExternalLink,
  Monitor, Smartphone, RefreshCw, CheckCircle2, Pencil, MousePointer2,
} from 'lucide-react'
import { useProjectContext } from '@/contexts/supabase-context'
import { SectionEditor } from './components/section-editor'
import { usePage } from './hooks/use-page'
import { cn } from '@/lib/utils'
import api from '@/lib/axios'
import type { PageSection } from '@/types'

function formatSectionLabel(type: string): string {
  const acronyms = ['cta', 'seo', 'faq']
  return type
    .split(/[_-]/)
    .map(w => acronyms.includes(w.toLowerCase()) ? w.toUpperCase() : w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ')
}

export function PageEditorPage() {
  const { slug } = useParams<{ slug: string }>()
  const navigate = useNavigate()
  const { activeProject } = useProjectContext()
  const queryClient = useQueryClient()
  const iframeRef = useRef<HTMLIFrameElement>(null)

  const { data: page, isLoading, isError } = usePage(slug!)

  const [localSections, setLocalSections] = useState<PageSection[]>([])
  const [activeKey, setActiveKey] = useState<string | null>(null)
  const [editMode, setEditMode] = useState(false)
  const [isPublishing, setIsPublishing] = useState(false)
  const [isMobile, setIsMobile] = useState(false)
  const [iframeKey, setIframeKey] = useState(0)
  const [iframeLoading, setIframeLoading] = useState(true)

  const editModeRef = useRef(editMode)
  useEffect(() => { editModeRef.current = editMode }, [editMode])

  const skipScrollRef = useRef(false)

  // Initialise from DB on page load
  useEffect(() => {
    if (page?.sections) {
      setLocalSections(page.sections)
    }
  }, [page?.id])

  // Send live preview to iframe on every section change (debounced to avoid flooding)
  useEffect(() => {
    const timer = setTimeout(() => sendPreviewUpdate(localSections), 80)
    return () => clearTimeout(timer)
  }, [localSections])

  // Scroll to section in iframe when activeKey changes (but not when set by iframe)
  useEffect(() => {
    if (skipScrollRef.current) { skipScrollRef.current = false; return }
    if (!activeKey) return
    sendScrollTo(activeKey)
  }, [activeKey])

  // Tell iframe about mode changes
  useEffect(() => {
    sendModeMessage(editMode ? 'edit' : 'preview')
    if (!editMode) { skipScrollRef.current = true; setActiveKey(null) }
  }, [editMode])

  // Listen for section clicks and scroll events from the iframe
  useEffect(() => {
    function handleIframeMessage(event: MessageEvent) {
      if (!event.data || typeof event.data !== 'object') return
      const { type, section } = event.data as { type: string; section: unknown }
      if (
        (type === 'PORTAL_SECTION_CLICK' || type === 'PORTAL_SECTION_VISIBLE') &&
        typeof section === 'string' &&
        section.length > 0 &&
        section.length <= 64
      ) {
        if (!editModeRef.current) return
        skipScrollRef.current = true
        setActiveKey(section)
      }
    }
    window.addEventListener('message', handleIframeMessage)
    return () => window.removeEventListener('message', handleIframeMessage)
  }, [])

  // Derived state
  const activeSection = activeKey
    ? (localSections.find(s => s.type === activeKey) ?? null)
    : null

  // Strip type from value passed into editor; editor returns data without type
  const activeSectionData = activeSection
    ? Object.fromEntries(Object.entries(activeSection).filter(([k]) => k !== 'type'))
    : null

  const dirtyKeys = useMemo(() => {
    const original = page?.sections ?? []
    return localSections
      .filter((s, i) => JSON.stringify(s) !== JSON.stringify(original[i]))
      .map(s => s.type)
  }, [localSections, page?.sections])

  const isDirty = dirtyKeys.length > 0

  function getOrigin() {
    try { return activeProject.site_url ? new URL(activeProject.site_url).origin : '*' }
    catch { return '*' }
  }

  function sendScrollTo(sectionType: string) {
    iframeRef.current?.contentWindow?.postMessage(
      { type: 'PORTAL_SCROLL_TO', section: sectionType },
      getOrigin(),
    )
  }

  function sendModeMessage(mode: 'edit' | 'preview') {
    iframeRef.current?.contentWindow?.postMessage(
      { type: 'PORTAL_SET_MODE', mode },
      getOrigin(),
    )
  }

  function sendPreviewUpdate(sections: PageSection[]) {
    iframeRef.current?.contentWindow?.postMessage(
      { type: 'PORTAL_UPDATE_SECTION', sections },
      getOrigin(),
    )
  }

  function handleSectionChange(updated: Record<string, unknown>) {
    if (!activeKey) return
    setLocalSections(prev =>
      prev.map(s => s.type === activeKey ? { type: activeKey, ...updated } : s)
    )
  }

  function handleDiscard() {
    if (!page?.sections) return
    setLocalSections(page.sections)
  }

  async function handlePublish() {
    if (!slug) return
    setIsPublishing(true)
    try {
      await api.put(`/api/cms/pages/${slug}/sections`, { sections: localSections })
      queryClient.invalidateQueries({ queryKey: ['page', slug] })
      queryClient.invalidateQueries({ queryKey: ['pages'] })
      setTimeout(() => { setIframeLoading(true); setIframeKey(k => k + 1) }, 1500)
    } finally {
      setIsPublishing(false)
    }
  }

  async function handleToggleVisibility() {
    if (!page || !slug) return
    setIsPublishing(true)
    try {
      await api.put(`/api/cms/pages/${slug}/visibility`, { is_published: !page.is_published })
      queryClient.invalidateQueries({ queryKey: ['page', slug] })
      queryClient.invalidateQueries({ queryKey: ['pages'] })
    } finally {
      setIsPublishing(false)
    }
  }

  function handleIframeLoad() {
    setIframeLoading(false)
    setTimeout(() => {
      sendModeMessage(editModeRef.current ? 'edit' : 'preview')
      sendPreviewUpdate(localSections)
      if (editModeRef.current && activeKey) sendScrollTo(activeKey)
    }, 300)
  }

  const siteURL = activeProject.site_url?.replace(/\/$/, '') ?? ''
  const pageURL = siteURL ? (slug === 'home' ? siteURL : `${siteURL}/${slug}`) : ''

  if (isLoading) {
    return (
      <div className="flex h-full animate-pulse">
        <div className="w-72 h-full bg-ink/5 border-r-2 border-ink/10" />
        <div className="flex-1 bg-ink/3" />
      </div>
    )
  }

  if (isError || !page) {
    return (
      <div className="flex items-center justify-center h-full gap-3 text-brand-red">
        <AlertCircle size={16} />
        <p className="font-mono text-sm">Failed to load page.</p>
      </div>
    )
  }

  // ── Shared toolbar controls ────────────────────────────────────────────────
  const previewControls = pageURL ? (
    <div className="flex items-center gap-1 shrink-0">
      <button
        type="button"
        onClick={() => { setIframeLoading(true); setIframeKey(k => k + 1) }}
        className="p-1.5 text-ink/40 hover:text-ink transition rounded"
        title="Refresh preview"
      >
        <RefreshCw size={13} />
      </button>
      <a
        href={pageURL}
        target="_blank"
        rel="noreferrer"
        className="p-1.5 text-ink/40 hover:text-ink transition rounded"
        title="Open live page"
      >
        <ExternalLink size={13} />
      </a>
      <div className="w-px h-4 bg-ink/15 mx-1" />
      <button
        type="button"
        onClick={() => setIsMobile(false)}
        className={cn('p-1.5 rounded transition', !isMobile ? 'text-ink' : 'text-ink/30 hover:text-ink/60')}
        title="Desktop view"
      >
        <Monitor size={14} />
      </button>
      <button
        type="button"
        onClick={() => setIsMobile(true)}
        className={cn('p-1.5 rounded transition', isMobile ? 'text-ink' : 'text-ink/30 hover:text-ink/60')}
        title="Mobile view"
      >
        <Smartphone size={14} />
      </button>
    </div>
  ) : null

  // ── Shared iframe area ─────────────────────────────────────────────────────
  const iframeArea = (
    <div className="flex-1 overflow-hidden flex items-start justify-center p-4">
      {pageURL ? (
        <div className={cn(
          'relative h-full bg-white border-2 border-ink/15 rounded-xl overflow-hidden',
          'shadow-[4px_4px_0_rgba(28,28,26,0.08)] transition-all duration-300',
          isMobile ? 'w-97.5' : 'w-full',
        )}>
          {iframeLoading && (
            <div className="absolute inset-0 flex items-center justify-center bg-white z-10">
              <div className="w-5 h-5 rounded-full border-2 border-ink border-t-transparent animate-spin" />
            </div>
          )}
          <iframe
            key={iframeKey}
            ref={iframeRef}
            src={pageURL}
            className="w-full h-full border-0"
            title={`Preview: ${page.title}`}
            onLoad={handleIframeLoad}
          />
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center gap-4 h-full text-ink/25">
          <Monitor size={48} strokeWidth={1} />
          <div className="text-center">
            <p className="font-sans font-semibold text-sm text-ink/40">No site URL configured</p>
            <p className="font-mono text-xs mt-1 text-ink/30">
              In Agency Hub → Project → <strong className="text-ink/40">Deploy Status</strong>, set and save the site URL
            </p>
          </div>
        </div>
      )}
    </div>
  )

  // ── PREVIEW MODE ──────────────────────────────────────────────────────────
  if (!editMode) {
    return (
      <div className="flex h-full flex-col overflow-hidden">
        <div className="h-11 shrink-0 border-b-2 border-ink/10 bg-white flex items-center gap-2 px-3">
          <button
            type="button"
            onClick={() => navigate('/pages')}
            className="p-1 text-ink/40 hover:text-ink transition rounded"
            aria-label="Back to pages"
          >
            <ChevronLeft size={15} />
          </button>
          <div className="min-w-0 flex-1 flex items-center gap-2">
            <p className="font-sans font-bold text-sm text-ink truncate">{page.title}</p>
            <div className="flex items-center gap-1.5 shrink-0">
              <span className={cn('w-1.5 h-1.5 rounded-full', page.is_published ? 'bg-forest' : 'bg-ink/25')} />
              <span className="font-mono text-[0.55rem] uppercase tracking-widest text-ink/40">
                {page.is_published ? 'Published' : 'Draft'}
              </span>
            </div>
          </div>
          <button
            type="button"
            onClick={() => setEditMode(true)}
            className="flex items-center gap-1.5 h-7 px-3 rounded-lg bg-ink text-white font-mono text-xs hover:bg-ink/80 transition shrink-0"
          >
            <Pencil size={11} />
            EDIT
          </button>
          {previewControls && (
            <>
              <div className="w-px h-4 bg-ink/15" />
              {previewControls}
            </>
          )}
        </div>
        <div className="flex-1 min-h-0 flex flex-col bg-ink/3">
          {iframeArea}
        </div>
      </div>
    )
  }

  // ── EDIT MODE ─────────────────────────────────────────────────────────────
  return (
    <div className="flex h-full overflow-hidden">

      {/* ── Left sidebar ── */}
      <div className="w-72 shrink-0 flex flex-col border-r-2 border-ink/10 bg-white overflow-hidden">

        {/* Sidebar header */}
        <div className="h-11 shrink-0 border-b-2 border-ink/10 flex items-center gap-2 px-3">
          <button
            type="button"
            onClick={() => navigate('/pages')}
            className="p-1 text-ink/40 hover:text-ink transition rounded"
            aria-label="Back to pages"
          >
            <ChevronLeft size={15} />
          </button>
          <div className="min-w-0 flex-1">
            <p className="font-sans font-bold text-sm text-ink truncate leading-tight">{page.title}</p>
            <div className="flex items-center gap-1.5">
              <span className={cn('w-1.5 h-1.5 rounded-full', page.is_published ? 'bg-forest' : 'bg-ink/25')} />
              <span className="font-mono text-[0.55rem] uppercase tracking-widest text-ink/40 leading-none">
                {page.is_published ? 'Published' : 'Draft'}
              </span>
            </div>
          </div>
        </div>

        {/* Main panel */}
        <div className="flex-1 overflow-y-auto">
          {activeKey && activeSectionData !== null ? (
            /* ── Field editor ── */
            <>
              <div className="flex items-center gap-2 px-3 py-2.5 border-b border-ink/8">
                <button
                  type="button"
                  onClick={() => { skipScrollRef.current = true; setActiveKey(null) }}
                  className="flex items-center gap-1 text-ink/40 hover:text-ink transition text-xs font-mono"
                >
                  <ChevronLeft size={12} />
                  Back
                </button>
                <span className="text-ink/20 text-xs">/</span>
                <span className="font-mono text-[0.6rem] uppercase tracking-widest text-ink/60 flex-1 truncate">
                  {formatSectionLabel(activeKey)}
                </span>
                {dirtyKeys.includes(activeKey) && (
                  <span className="w-1.5 h-1.5 rounded-full bg-amber-500 shrink-0" />
                )}
              </div>
              <div className="px-3 py-4">
                <SectionEditor
                  sectionKey={activeKey}
                  value={activeSectionData}
                  onChange={handleSectionChange}
                />
              </div>
            </>
          ) : (
            /* ── Click-to-select empty state ── */
            <div className="flex flex-col h-full">
              <div className="flex-1 flex flex-col items-center justify-center gap-3 px-6 text-center">
                <div className="w-10 h-10 rounded-full bg-ink/5 flex items-center justify-center">
                  <MousePointer2 size={18} className="text-ink/30" />
                </div>
                <p className="font-sans text-sm font-medium text-ink/50 leading-snug">
                  Click any section in the preview to edit it
                </p>
              </div>

              {localSections.length > 0 && (
                <div className="shrink-0 border-t border-ink/8 px-3 py-3">
                  <p className="font-mono text-[0.5rem] uppercase tracking-widest text-ink/25 mb-2 px-1">
                    Jump to section
                  </p>
                  <div className="flex flex-col gap-0.5">
                    {localSections.map(s => (
                      <button
                        key={s.type}
                        type="button"
                        onClick={() => { setActiveKey(s.type); sendScrollTo(s.type) }}
                        className="flex items-center gap-2 px-2 py-1.5 rounded-md text-left hover:bg-ink/5 transition-colors group"
                      >
                        {dirtyKeys.includes(s.type) ? (
                          <span className="w-1 h-1 rounded-full bg-amber-400 shrink-0" />
                        ) : (
                          <span className="w-1 h-1 rounded-full bg-ink/15 group-hover:bg-ink/30 shrink-0 transition-colors" />
                        )}
                        <span className="font-mono text-[0.65rem] text-ink/40 group-hover:text-ink/70 transition-colors">
                          {formatSectionLabel(s.type)}
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* ── Publish bar ── */}
        <div className="shrink-0 border-t-2 border-ink/10 p-3 bg-white flex flex-col gap-2.5">
          {isDirty ? (
            <>
              <p className="font-mono text-[0.6rem] text-amber-600 leading-tight">
                {dirtyKeys.length} section{dirtyKeys.length > 1 ? 's' : ''} with unpublished changes
              </p>
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={handleDiscard}
                  disabled={isPublishing}
                  className="flex-1 border-2 border-ink/20 rounded-lg py-2 font-mono text-xs text-ink/50 hover:border-ink/50 hover:text-ink transition disabled:opacity-40"
                >
                  Discard
                </button>
                <button
                  type="button"
                  onClick={handlePublish}
                  disabled={isPublishing}
                  className="flex-1 bg-forest border-2 border-forest text-white rounded-lg py-2 font-mono text-xs font-bold uppercase tracking-widest hover:bg-forest/90 transition disabled:opacity-50"
                >
                  {isPublishing ? '…' : 'Publish'}
                </button>
              </div>
            </>
          ) : (
            <div className="flex items-center gap-2 text-ink/35 py-0.5">
              <CheckCircle2 size={13} />
              <span className="font-mono text-xs">All changes published</span>
            </div>
          )}
          <div className="border-t border-ink/8 pt-2">
            <button
              type="button"
              onClick={handleToggleVisibility}
              disabled={isPublishing}
              className="font-mono text-[0.65rem] text-ink/40 hover:text-ink transition disabled:opacity-40"
            >
              {page.is_published ? '↓ Unpublish page' : '↑ Make page visible on site'}
            </button>
          </div>
        </div>
      </div>

      {/* ── Right: live preview ── */}
      <div className="flex-1 min-w-0 flex flex-col bg-ink/3 overflow-hidden">
        <div className="h-11 shrink-0 border-b-2 border-ink/10 bg-white flex items-center gap-2 px-3">
          <div className="flex-1 min-w-0 bg-ink/5 rounded-md px-3 py-1.5">
            <span className="font-mono text-[0.65rem] text-ink/50 truncate block">
              {pageURL || 'No site URL — add it in Agency Hub → Project → Deploy Status'}
            </span>
          </div>
          <button
            type="button"
            onClick={() => setEditMode(false)}
            className="flex items-center gap-1.5 h-7 px-3 rounded-lg border-2 border-ink/20 font-mono text-xs text-ink/50 hover:border-ink/50 hover:text-ink transition shrink-0"
          >
            <Monitor size={11} />
            PREVIEW
          </button>
          {previewControls}
        </div>
        {iframeArea}
      </div>
    </div>
  )
}
