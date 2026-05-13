import { Link } from 'react-router'
import { FileText, PenLine } from 'lucide-react'
import { useRecentEdits } from '../hooks/use-recent-edits'

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

function EditSkeleton() {
  return (
    <div className="flex items-center gap-3 px-4 py-3">
      <div className="w-6 h-6 rounded-md bg-ink/8 shrink-0 animate-pulse" />
      <div className="flex-1 flex flex-col gap-1.5">
        <div className="h-3 w-40 rounded bg-ink/8 animate-pulse" />
        <div className="h-2.5 w-24 rounded bg-ink/5 animate-pulse" />
      </div>
      <div className="h-2.5 w-12 rounded bg-ink/5 animate-pulse" />
    </div>
  )
}

export function RecentEdits() {
  const { data: edits = [], isLoading } = useRecentEdits()

  return (
    <div>
      <h2 className="font-mono text-[0.65rem] uppercase tracking-widest text-ink/40 mb-3">Recent edits</h2>

      <div className="border-2 border-ink/10 rounded-xl overflow-hidden divide-y-2 divide-ink/8">
        {isLoading ? (
          Array.from({ length: 4 }).map((_, i) => <EditSkeleton key={i} />)
        ) : edits.length === 0 ? (
          <div className="px-4 py-8 text-center">
            <p className="font-mono text-[0.65rem] text-ink/30">No content yet.</p>
          </div>
        ) : (
          edits.map((edit) => {
            const href = edit.type === 'page' ? `/pages/${edit.slug}` : `/blog/${edit.id}/edit`
            const Icon = edit.type === 'page' ? FileText : PenLine
            return (
              <Link
                key={`${edit.type}-${edit.id}`}
                to={href}
                className="flex items-center gap-3 px-4 py-3 hover:bg-ink/[0.02] transition group"
              >
                <div className="rounded-md bg-ink/5 p-1.5 shrink-0 group-hover:bg-ink/10 transition">
                  <Icon size={12} className="text-ink/50" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="font-mono text-xs text-ink truncate">{edit.title}</p>
                  <p className="font-mono text-[0.6rem] text-ink/35 mt-0.5 capitalize">{edit.type}</p>
                </div>
                <span className="font-mono text-[0.6rem] text-ink/30 shrink-0">{relativeTime(edit.updated_at)}</span>
              </Link>
            )
          })
        )}
      </div>
    </div>
  )
}
