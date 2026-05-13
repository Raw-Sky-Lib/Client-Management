import { useState } from 'react'
import { Inbox, AlertCircle } from 'lucide-react'
import { useSubmissions } from './hooks/use-submissions'
import { SubmissionsTable, SkeletonRow } from './components/submissions-table'
import { SubmissionDetail } from './components/submission-detail'
import type { FormSubmission } from '@/types'

export function FormsPage() {
  const { data: submissions, isLoading, isError } = useSubmissions()
  const [open, setOpen] = useState<FormSubmission | null>(null)

  const unreadCount = (submissions ?? []).filter((s) => !s.is_read).length

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="font-sans font-extrabold text-2xl tracking-tight text-ink flex items-center gap-2">
            <Inbox size={20} className="text-ink/60" />
            Forms
            {unreadCount > 0 && (
              <span className="inline-flex items-center justify-center min-w-5 h-5 px-1.5 rounded-full bg-forest text-white font-mono text-[0.6rem] font-bold">
                {unreadCount}
              </span>
            )}
          </h2>
          <p className="font-mono text-xs text-ink/50 mt-0.5">
            {submissions ? `${submissions.length} submission${submissions.length !== 1 ? 's' : ''}` : 'Loading…'}
          </p>
        </div>
      </div>

      {/* Loading */}
      {isLoading && (
        <div className="border-2 border-ink rounded-xl overflow-hidden bg-white shadow-hard-xs">
          <div className="grid grid-cols-[16px_1fr_1fr_140px] gap-4 items-center px-5 py-3 bg-ink border-b-2 border-ink">
            <div /><div /><div /><div />
          </div>
          {Array.from({ length: 5 }).map((_, i) => <SkeletonRow key={i} />)}
        </div>
      )}

      {/* Error */}
      {isError && (
        <div className="flex items-center gap-3 text-brand-red">
          <AlertCircle size={16} className="shrink-0" />
          <p className="font-mono text-sm">Failed to load submissions. Check your connection and try again.</p>
        </div>
      )}

      {/* Table */}
      {!isLoading && !isError && submissions && (
        <SubmissionsTable submissions={submissions} onOpen={setOpen} />
      )}

      {/* Detail drawer */}
      <SubmissionDetail submission={open} onClose={() => setOpen(null)} />
    </div>
  )
}
