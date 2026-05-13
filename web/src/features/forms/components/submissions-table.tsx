import { cn, formatDate } from '@/lib/utils'
import type { FormSubmission } from '@/types'

interface SubmissionsTableProps {
  submissions: FormSubmission[]
  onOpen: (submission: FormSubmission) => void
}

function dataPreview(data: Record<string, unknown>): string {
  return Object.entries(data)
    .slice(0, 2)
    .map(([k, v]) => `${k}: ${String(v)}`)
    .join(' · ')
}

function SkeletonRow() {
  return (
    <div className="flex items-center gap-4 px-5 py-4 border-b-2 border-ink/6 animate-pulse">
      <div className="w-2 h-2 rounded-full bg-ink/10 shrink-0" />
      <div className="flex-1 flex flex-col gap-1.5">
        <div className="h-3 w-24 bg-ink/10 rounded" />
        <div className="h-2.5 w-48 bg-ink/6 rounded" />
      </div>
      <div className="h-2.5 w-20 bg-ink/6 rounded" />
    </div>
  )
}

export function SubmissionsTable({ submissions, onOpen }: SubmissionsTableProps) {
  return (
    <div className="border-2 border-ink rounded-xl overflow-hidden bg-white shadow-hard-xs">
      {/* Table header */}
      <div className="grid grid-cols-[16px_1fr_1fr_140px] gap-4 items-center px-5 py-3 bg-ink border-b-2 border-ink">
        <div />
        <span className="font-mono text-[0.65rem] uppercase tracking-widest text-white/50">Form</span>
        <span className="font-mono text-[0.65rem] uppercase tracking-widest text-white/50">Preview</span>
        <span className="font-mono text-[0.65rem] uppercase tracking-widest text-white/50 text-right">Submitted</span>
      </div>

      {submissions.length === 0 && (
        <div className="px-5 py-16 text-center">
          <p className="font-sans font-semibold text-ink/30">No submissions yet</p>
          <p className="font-mono text-xs text-ink/25 mt-1">Submissions from your site forms will appear here.</p>
        </div>
      )}

      {submissions.map((s) => (
        <button
          key={s.id}
          type="button"
          onClick={() => onOpen(s)}
          className={cn(
            'w-full grid grid-cols-[16px_1fr_1fr_140px] gap-4 items-center px-5 py-4',
            'border-b-2 border-ink/6 last:border-0 text-left transition-colors',
            s.is_read ? 'hover:bg-ink/3' : 'bg-forest/4 hover:bg-forest/7',
          )}
        >
          {/* Unread dot */}
          <div className="flex items-center justify-center">
            {!s.is_read && (
              <span className="w-2 h-2 rounded-full bg-forest shrink-0" />
            )}
          </div>

          {/* Form name */}
          <span className={cn(
            'font-sans text-sm truncate',
            s.is_read ? 'text-ink/70 font-normal' : 'text-ink font-semibold',
          )}>
            {s.form_name}
          </span>

          {/* Data preview */}
          <span className="font-mono text-xs text-ink/40 truncate">
            {dataPreview(s.data)}
          </span>

          {/* Date */}
          <span className="font-mono text-xs text-ink/40 text-right shrink-0">
            {formatDate(s.submitted_at)}
          </span>
        </button>
      ))}
    </div>
  )
}

export { SkeletonRow }
