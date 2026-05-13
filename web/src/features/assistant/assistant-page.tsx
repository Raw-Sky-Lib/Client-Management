import { useState, useCallback } from 'react'
import { Sparkles } from 'lucide-react'
import { toast } from 'sonner'
import { useTenantSupabase } from '@/contexts/supabase-context'
import { useQueryClient } from '@tanstack/react-query'
import { useAssistant, type RateLimitError } from './hooks/use-assistant'
import { InstructionForm } from './components/instruction-form'
import { DiffPreview } from './components/diff-preview'
import { ApplyBar } from './components/apply-bar'
import { RateLimitBanner } from './components/rate-limit-banner'
import type { FieldChange, GenerateRequest } from '@/types'
import type { SaveState } from '@/components/shared/save-indicator'
import api from '@/lib/axios'

type AssistantState = 'idle' | 'preview' | 'applied'

function isRateLimitError(err: unknown): err is RateLimitError {
  return typeof err === 'object' && err !== null && 'type' in err && 'message' in err
}

export function AssistantPage() {
  const supabase = useTenantSupabase()
  const queryClient = useQueryClient()
  const { mutateAsync: generate, isPending: isGenerating } = useAssistant()

  const [form, setForm] = useState<GenerateRequest & { instruction: string }>({
    page_slug: '',
    section_type: '',
    instruction: '',
  })

  const [assistantState, setAssistantState] = useState<AssistantState>('idle')
  const [changes, setChanges] = useState<FieldChange[]>([])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [rateLimitError, setRateLimitError] = useState<RateLimitError | null>(null)
  const [saveState, setSaveState] = useState<SaveState>('idle')

  function handleFormChange(patch: Partial<typeof form>) {
    setForm((prev) => ({ ...prev, ...patch }))
    // Navigating away from preview resets it
    if ('page_slug' in patch || 'section_type' in patch) {
      setAssistantState('idle')
      setChanges([])
      setSelected(new Set())
      setRateLimitError(null)
    }
  }

  async function handleGenerate() {
    if (!form.page_slug || !form.section_type || !form.instruction.trim()) return
    setRateLimitError(null)
    setAssistantState('idle')

    try {
      const result = await generate({
        page_slug: form.page_slug,
        section_type: form.section_type,
        instruction: form.instruction,
      })

      if (!result.length) {
        toast.info('No suggestions returned. Try a more specific instruction.')
        return
      }

      setChanges(result)
      setSelected(new Set(result.map((c) => c.field)))
      setAssistantState('preview')
    } catch (err) {
      if (isRateLimitError(err)) {
        setRateLimitError(err)
      } else {
        toast.error(err instanceof Error ? err.message : 'The assistant is temporarily unavailable.')
      }
    }
  }

  const handleToggle = useCallback((field: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      next.has(field) ? next.delete(field) : next.add(field)
      return next
    })
  }, [])

  const handleToggleAll = useCallback((checked: boolean) => {
    setSelected(checked ? new Set(changes.map((c) => c.field)) : new Set())
  }, [changes])

  async function handleApply() {
    if (selected.size === 0) return
    setSaveState('saving')

    try {
      // Fetch current page to avoid overwriting concurrent edits
      const { data: page, error: fetchErr } = await supabase
        .from('pages')
        .select('sections')
        .eq('slug', form.page_slug)
        .single()
      if (fetchErr || !page) throw new Error('Could not load page.')

      const sections = { ...(page.sections as Record<string, Record<string, unknown>>) }
      const section = { ...(sections[form.section_type] ?? {}) }

      for (const change of changes) {
        if (selected.has(change.field)) {
          section[change.field] = change.proposed
        }
      }
      sections[form.section_type] = section

      const { error: saveErr } = await supabase
        .from('pages')
        .update({ sections, updated_at: new Date().toISOString() })
        .eq('slug', form.page_slug)
      if (saveErr) throw saveErr

      // Trigger ISR — fire and forget
      api.post('/api/revalidate', { paths: [`/${form.page_slug === 'home' ? '' : form.page_slug}`] }).catch(() => null)

      queryClient.invalidateQueries({ queryKey: ['page', form.page_slug] })
      queryClient.invalidateQueries({ queryKey: ['pages'] })

      setSaveState('saved')
      setAssistantState('applied')
      setTimeout(() => {
        setSaveState('idle')
        setAssistantState('idle')
        setChanges([])
        setSelected(new Set())
        setForm((prev) => ({ ...prev, instruction: '' }))
      }, 2500)
    } catch (err) {
      setSaveState('error')
      toast.error(err instanceof Error ? err.message : 'Failed to apply changes.')
      setTimeout(() => setSaveState('idle'), 3000)
    }
  }

  function handleDiscard() {
    setAssistantState('idle')
    setChanges([])
    setSelected(new Set())
    setSaveState('idle')
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <div>
        <h2 className="font-sans font-extrabold text-2xl tracking-tight text-ink flex items-center gap-2">
          <Sparkles size={20} className="text-forest" />
          Assistant
        </h2>
        <p className="font-mono text-xs text-ink/50 mt-0.5">
          Describe a change — review the diff — apply what you want.
        </p>
      </div>

      {/* Main layout */}
      <div className="flex gap-6 items-start">

        {/* Left: instruction form */}
        <div
          className="w-72 shrink-0 border-2 border-ink rounded-xl overflow-hidden bg-white shadow-hard-xs"
        >
          <div className="px-4 py-3 bg-ink border-b-2 border-ink">
            <span className="font-mono text-[0.65rem] uppercase tracking-widest text-white/60">
              Instruction
            </span>
          </div>
          <div className="p-4 flex flex-col gap-4">
            <InstructionForm
              value={form}
              onChange={handleFormChange}
              onSubmit={handleGenerate}
              isLoading={isGenerating}
              disabled={assistantState === 'applied'}
            />
            {rateLimitError && (
              <RateLimitBanner type={rateLimitError.type} message={rateLimitError.message} />
            )}
          </div>
        </div>

        {/* Right: diff preview or empty state */}
        <div className="flex-1 min-w-0 border-2 border-ink rounded-xl overflow-hidden bg-white shadow-hard-xs">
          {assistantState === 'idle' && (
            <div className="flex flex-col items-center justify-center py-24 px-8 text-center gap-3">
              <Sparkles size={36} className="text-ink/15" />
              <p className="font-sans font-semibold text-ink/40">No suggestions yet</p>
              <p className="font-mono text-xs text-ink/30 max-w-xs">
                Pick a page and section, describe what to improve, and hit Generate.
              </p>
            </div>
          )}

          {(assistantState === 'preview' || assistantState === 'applied') && changes.length > 0 && (
            <>
              <DiffPreview
                changes={changes}
                selected={selected}
                onToggle={handleToggle}
                onToggleAll={handleToggleAll}
              />
              <ApplyBar
                selectedCount={selected.size}
                saveState={saveState}
                onApply={handleApply}
                onDiscard={handleDiscard}
              />
            </>
          )}
        </div>
      </div>
    </div>
  )
}
