import { cn } from '@/lib/utils'
import type { FieldChange } from '@/types'

interface DiffPreviewProps {
  changes: FieldChange[]
  selected: Set<string>
  onToggle: (field: string) => void
  onToggleAll: (checked: boolean) => void
}

function formatFieldLabel(field: string): string {
  return field
    .split(/[_-]/)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ')
}

export function DiffPreview({ changes, selected, onToggle, onToggleAll }: DiffPreviewProps) {
  const allSelected = changes.every((c) => selected.has(c.field))
  const someSelected = changes.some((c) => selected.has(c.field))

  return (
    <div className="flex flex-col">
      {/* Header */}
      <div className="px-4 py-3 bg-ink border-b-2 border-ink flex items-center gap-3">
        <input
          type="checkbox"
          checked={allSelected}
          ref={(el) => { if (el) el.indeterminate = someSelected && !allSelected }}
          onChange={(e) => onToggleAll(e.target.checked)}
          className="accent-forest w-3.5 h-3.5 shrink-0"
          aria-label="Select all changes"
        />
        <span className="font-mono text-[0.65rem] uppercase tracking-widest text-white/60">
          {selected.size} of {changes.length} change{changes.length !== 1 ? 's' : ''} selected
        </span>
      </div>

      {/* Rows */}
      <div className="divide-y-2 divide-ink/6">
        {changes.map((change) => {
          const isSelected = selected.has(change.field)
          return (
            <div
              key={change.field}
              onClick={() => onToggle(change.field)}
              className={cn(
                'flex items-start gap-3 px-4 py-4 cursor-pointer transition-colors',
                isSelected ? 'bg-white hover:bg-ink/2' : 'bg-ink/3 hover:bg-ink/5',
              )}
            >
              <input
                type="checkbox"
                checked={isSelected}
                onChange={() => onToggle(change.field)}
                onClick={(e) => e.stopPropagation()}
                className="accent-forest w-3.5 h-3.5 shrink-0 mt-0.5"
                aria-label={`Select ${change.field}`}
              />

              <div className="flex-1 min-w-0 flex flex-col gap-2.5">
                {/* Field label + notes */}
                <div className="flex items-baseline gap-2 flex-wrap">
                  <span className="font-mono text-xs font-bold text-ink">
                    {formatFieldLabel(change.field)}
                  </span>
                  {change.notes && (
                    <span className="font-mono text-[0.65rem] text-ink/40 italic">
                      {change.notes}
                    </span>
                  )}
                </div>

                {/* Current → Proposed */}
                <div className="grid grid-cols-2 gap-3">
                  <div className="flex flex-col gap-1">
                    <span className="font-mono text-[0.6rem] uppercase tracking-widest text-ink/30">Current</span>
                    <p className="font-mono text-xs text-ink/60 bg-ink/4 rounded-lg px-3 py-2 leading-relaxed whitespace-pre-wrap break-words">
                      {change.current || <span className="italic text-ink/30">empty</span>}
                    </p>
                  </div>
                  <div className="flex flex-col gap-1">
                    <span className="font-mono text-[0.6rem] uppercase tracking-widest text-forest/60">Proposed</span>
                    <p className={cn(
                      'font-mono text-xs rounded-lg px-3 py-2 leading-relaxed whitespace-pre-wrap break-words',
                      isSelected
                        ? 'text-forest bg-forest/8 border-2 border-forest/20'
                        : 'text-ink/40 bg-ink/4',
                    )}>
                      {change.proposed}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
