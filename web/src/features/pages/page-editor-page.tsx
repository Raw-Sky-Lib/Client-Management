import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router'
import { useQueryClient } from '@tanstack/react-query'
import { ChevronLeft, AlertCircle } from 'lucide-react'
import { useTenantSupabase } from '@/contexts/supabase-context'
import { SaveIndicator, type SaveState } from '@/components/shared/save-indicator'
import { SectionEditor } from './components/section-editor'
import { usePage } from './hooks/use-page'
import { cn } from '@/lib/utils'
import api from '@/lib/axios'

function formatSectionLabel(key: string): string {
  const acronyms = ['cta', 'seo', 'faq']
  return key
    .split(/[_-]/)
    .map(w => acronyms.includes(w.toLowerCase()) ? w.toUpperCase() : w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ')
}

export function PageEditorPage() {
  const { slug } = useParams<{ slug: string }>()
  const navigate = useNavigate()
  const supabase = useTenantSupabase()
  const queryClient = useQueryClient()

  const { data: page, isLoading, isError } = usePage(slug!)

  const [localSections, setLocalSections] = useState<Record<string, unknown>>({})
  const [activeKey, setActiveKey] = useState<string | null>(null)
  const [saveState, setSaveState] = useState<SaveState>('idle')
  const [isPublishing, setIsPublishing] = useState(false)

  useEffect(() => {
    if (page?.sections) {
      const sections = page.sections as Record<string, unknown>
      setLocalSections(sections)
      setActiveKey(prev => prev ?? Object.keys(sections)[0] ?? null)
    }
  }, [page?.id])

  const sectionKeys = Object.keys(localSections)
  const activeSection = activeKey ? (localSections[activeKey] as Record<string, unknown>) : null

  async function handleSave() {
    if (!slug) return
    setSaveState('saving')
    try {
      const { error } = await supabase
        .from('pages')
        .update({ sections: localSections, updated_at: new Date().toISOString() })
        .eq('slug', slug)
      if (error) throw error
      api.post('/api/revalidate', { paths: ['/'] }).catch(() => null)
      queryClient.invalidateQueries({ queryKey: ['page', slug] })
      queryClient.invalidateQueries({ queryKey: ['pages'] })
      setSaveState('saved')
      setTimeout(() => setSaveState('idle'), 2000)
    } catch {
      setSaveState('error')
      setTimeout(() => setSaveState('idle'), 3000)
    }
  }

  async function handleTogglePublish() {
    if (!page || !slug) return
    setIsPublishing(true)
    try {
      const { error } = await supabase
        .from('pages')
        .update({ is_published: !page.is_published })
        .eq('slug', slug)
      if (!error) {
        api.post('/api/revalidate', { paths: ['/'] }).catch(() => null)
        queryClient.invalidateQueries({ queryKey: ['page', slug] })
        queryClient.invalidateQueries({ queryKey: ['pages'] })
      }
    } finally {
      setIsPublishing(false)
    }
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-6 animate-pulse">
        <div className="h-8 w-64 bg-ink/10 rounded" />
        <div className="flex gap-6">
          <div className="w-44 h-64 bg-ink/10 rounded-xl" />
          <div className="flex-1 h-64 bg-ink/10 rounded-xl" />
        </div>
      </div>
    )
  }

  if (isError || !page) {
    return (
      <div className="flex items-center gap-3 text-brand-red">
        <AlertCircle size={16} />
        <p className="font-mono text-sm">Failed to load page. Check your connection and try again.</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">

      {/* Top bar */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3 min-w-0">
          <button
            type="button"
            onClick={() => navigate('/pages')}
            className="flex items-center gap-1 font-mono text-xs text-ink/50 hover:text-ink transition shrink-0"
          >
            <ChevronLeft size={14} />
            Pages
          </button>
          <span className="font-mono text-ink/25 shrink-0">/</span>
          <h2 className="font-sans font-extrabold text-xl tracking-tight text-ink truncate">
            {page.title}
          </h2>
        </div>

        <div className="flex items-center gap-3 shrink-0">
          <span className={cn(
            'inline-flex items-center gap-1.5 font-mono text-xs font-bold uppercase tracking-widest',
            page.is_published ? 'text-forest' : 'text-ink/40',
          )}>
            <span className={cn(
              'w-1.5 h-1.5 rounded-full shrink-0',
              page.is_published ? 'bg-forest' : 'bg-ink/25',
            )} />
            {page.is_published ? 'Published' : 'Draft'}
          </span>

          <button
            type="button"
            onClick={handleTogglePublish}
            disabled={isPublishing}
            className="border-2 border-ink rounded-lg px-4 py-2 font-mono text-xs font-bold uppercase tracking-widest text-ink hover:bg-ink hover:text-cream transition disabled:opacity-50"
          >
            {isPublishing ? '…' : page.is_published ? 'Unpublish' : 'Publish'}
          </button>
        </div>
      </div>

      {/* Editor layout */}
      <div className="flex gap-6 items-start">

        {/* Section list */}
        <div
          className="w-44 shrink-0 border-2 border-ink rounded-xl overflow-hidden bg-white sticky top-0"
          style={{ boxShadow: 'var(--shadow-hard-xs)' }}
        >
          <div className="px-4 py-3 bg-ink border-b-2 border-ink">
            <span className="font-mono text-[0.65rem] uppercase tracking-widest text-white/60">
              Sections
            </span>
          </div>

          {sectionKeys.length === 0 && (
            <p className="px-4 py-6 font-mono text-xs text-ink/40 text-center">No sections</p>
          )}

          {sectionKeys.map(key => (
            <button
              key={key}
              type="button"
              onClick={() => setActiveKey(key)}
              className={cn(
                'w-full text-left px-4 py-3 border-b-2 border-ink/8 last:border-0 font-sans text-sm transition-colors',
                activeKey === key
                  ? 'bg-ink text-white font-semibold'
                  : 'text-ink hover:bg-ink/5',
              )}
            >
              {formatSectionLabel(key)}
            </button>
          ))}
        </div>

        {/* Active section editor */}
        {activeKey && activeSection !== null ? (
          <div
            className="flex-1 min-w-0 border-2 border-ink rounded-xl overflow-hidden bg-white"
            style={{ boxShadow: 'var(--shadow-hard-xs)' }}
          >
            <div className="px-6 py-4 bg-ink border-b-2 border-ink">
              <h3 className="font-sans font-extrabold text-white">
                {formatSectionLabel(activeKey)}
              </h3>
            </div>

            <div className="p-6">
              <SectionEditor
                sectionKey={activeKey}
                value={activeSection}
                onChange={updated =>
                  setLocalSections(prev => ({ ...prev, [activeKey]: updated }))
                }
              />
            </div>

            <div className="px-6 py-4 border-t-2 border-ink/10 flex items-center justify-between">
              <SaveIndicator state={saveState} />
              <button
                type="button"
                onClick={handleSave}
                disabled={saveState === 'saving'}
                className="btn-ink-shadow flex items-center gap-2 bg-forest text-white font-sans font-extrabold text-sm uppercase tracking-wider py-3 px-6 rounded-lg border-2 border-ink disabled:opacity-60"
              >
                Save changes →
              </button>
            </div>
          </div>
        ) : (
          <div className="flex-1 flex items-center justify-center py-20 text-ink/30 font-mono text-sm">
            Select a section to edit
          </div>
        )}
      </div>
    </div>
  )
}
