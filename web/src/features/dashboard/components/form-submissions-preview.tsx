import { Link } from 'react-router'
import { ArrowRight, Inbox } from 'lucide-react'
import { useSubmissions } from '@/features/forms/hooks/use-submissions'

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const mins = Math.floor(diff / 60_000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  const days = Math.floor(hrs / 24)
  return `${days}d ago`
}

function dataPreview(data: Record<string, unknown>): string {
  const entries = Object.entries(data).slice(0, 2)
  return entries.map(([, v]) => String(v)).join(' · ')
}

export function FormSubmissionsPreview() {
  const { data: submissions = [] } = useSubmissions()
  const unread = submissions.filter((s) => !s.is_read)
  const preview = unread.slice(0, 3)

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h2 className="font-mono text-[0.65rem] uppercase tracking-widest text-ink/40">
          Submissions
          {unread.length > 0 && (
            <span className="ml-2 inline-flex items-center justify-center rounded-full bg-forest text-cream font-mono text-[0.55rem] px-1.5 py-0.5 leading-none">
              {unread.length}
            </span>
          )}
        </h2>
        <Link to="/forms" className="flex items-center gap-1 font-mono text-[0.6rem] text-ink/40 hover:text-ink transition">
          View all <ArrowRight size={10} />
        </Link>
      </div>

      {preview.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-2 border-2 border-ink/8 rounded-xl py-8 text-center">
          <Inbox size={18} className="text-ink/20" />
          <p className="font-mono text-[0.65rem] text-ink/30">No unread submissions</p>
        </div>
      ) : (
        <div className="flex flex-col border-2 border-ink/10 rounded-xl overflow-hidden divide-y-2 divide-ink/8">
          {preview.map((sub) => (
            <Link
              key={sub.id}
              to="/forms"
              className="flex items-start justify-between gap-3 px-4 py-3 hover:bg-ink/[0.02] transition"
            >
              <div className="min-w-0">
                <p className="font-mono text-xs font-bold text-ink truncate">{sub.form_name}</p>
                <p className="font-mono text-[0.6rem] text-ink/40 truncate mt-0.5">{dataPreview(sub.data)}</p>
              </div>
              <span className="font-mono text-[0.6rem] text-ink/30 shrink-0 mt-0.5">{relativeTime(sub.submitted_at)}</span>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
