import { Sparkles, Loader2 } from 'lucide-react'
import { usePages } from '@/features/pages/hooks/use-pages'
import { usePage } from '@/features/pages/hooks/use-page'
import { cn } from '@/lib/utils'
import type { GenerateRequest } from '@/types'

interface InstructionFormProps {
  value: Omit<GenerateRequest, 'instruction'> & { instruction: string }
  onChange: (v: Partial<GenerateRequest & { instruction: string }>) => void
  onSubmit: () => void
  isLoading: boolean
  disabled?: boolean
}

function formatSectionLabel(key: string): string {
  const acronyms = ['cta', 'seo', 'faq']
  return key
    .split(/[_-]/)
    .map((w) => (acronyms.includes(w.toLowerCase()) ? w.toUpperCase() : w.charAt(0).toUpperCase() + w.slice(1)))
    .join(' ')
}

const selectClass = [
  'w-full border-2 border-ink/15 rounded-lg px-3 py-2 font-mono text-xs text-ink',
  'bg-white focus:outline-none focus:border-ink transition appearance-none',
  'disabled:opacity-50 disabled:bg-ink/4',
].join(' ')

const labelClass = 'block font-mono text-[0.65rem] uppercase tracking-widest text-ink/50 mb-1.5'

export function InstructionForm({ value, onChange, onSubmit, isLoading, disabled }: InstructionFormProps) {
  const { data: pages = [], isLoading: pagesLoading } = usePages()
  const { data: page } = usePage(value.page_slug)

  const sectionKeys = page?.sections ? Object.keys(page.sections) : []
  const canSubmit = value.page_slug && value.section_type && value.instruction.trim().length > 0

  return (
    <div className="flex flex-col gap-4">
      {/* Page */}
      <div>
        <label className={labelClass}>Page</label>
        <div className="relative">
          <select
            value={value.page_slug}
            onChange={(e) => onChange({ page_slug: e.target.value, section_type: '' })}
            disabled={pagesLoading || disabled}
            className={selectClass}
          >
            <option value="">Select a page…</option>
            {pages.map((p) => (
              <option key={p.slug} value={p.slug}>{p.title}</option>
            ))}
          </select>
        </div>
      </div>

      {/* Section */}
      <div>
        <label className={labelClass}>Section</label>
        <select
          value={value.section_type}
          onChange={(e) => onChange({ section_type: e.target.value })}
          disabled={!value.page_slug || sectionKeys.length === 0 || disabled}
          className={selectClass}
        >
          <option value="">
            {!value.page_slug ? 'Select a page first' : sectionKeys.length === 0 ? 'No sections' : 'Select a section…'}
          </option>
          {sectionKeys.map((key) => (
            <option key={key} value={key}>{formatSectionLabel(key)}</option>
          ))}
        </select>
      </div>

      {/* Instruction */}
      <div>
        <label className={labelClass}>Instruction</label>
        <textarea
          value={value.instruction}
          onChange={(e) => onChange({ instruction: e.target.value })}
          disabled={disabled}
          placeholder="e.g. Make the headline more compelling and action-oriented…"
          rows={5}
          className={cn(
            'w-full border-2 border-ink/15 rounded-lg px-3 py-2.5 font-mono text-xs text-ink',
            'placeholder:text-ink/25 focus:outline-none focus:border-ink transition resize-none',
            'disabled:opacity-50 disabled:bg-ink/4',
          )}
        />
      </div>

      {/* Generate */}
      <button
        type="button"
        onClick={onSubmit}
        disabled={!canSubmit || isLoading || disabled}
        className={cn(
          'flex items-center justify-center gap-2 w-full border-2 border-ink rounded-xl px-4 py-3',
          'font-mono text-xs font-bold uppercase tracking-widest transition',
          canSubmit && !isLoading && !disabled
            ? 'bg-forest text-white border-forest hover:bg-forest-deep'
            : 'opacity-40 cursor-not-allowed text-ink',
        )}
      >
        {isLoading ? (
          <>
            <Loader2 size={13} className="animate-spin" />
            Generating…
          </>
        ) : (
          <>
            <Sparkles size={13} />
            Generate suggestions
          </>
        )}
      </button>
    </div>
  )
}
