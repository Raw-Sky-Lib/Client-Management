import { Link } from 'react-router'
import { FileText, AlertCircle } from 'lucide-react'
import { usePages } from './hooks/use-pages'
import { formatDate } from '@/lib/utils'

function StatusBadge({ published }: { published: boolean }) {
  return published ? (
    <span className="inline-flex items-center gap-1.5 font-mono text-xs font-bold uppercase tracking-widest text-forest">
      <span className="w-1.5 h-1.5 rounded-full bg-forest shrink-0" />
      Published
    </span>
  ) : (
    <span className="inline-flex items-center gap-1.5 font-mono text-xs font-bold uppercase tracking-widest text-ink/40">
      <span className="w-1.5 h-1.5 rounded-full bg-ink/25 shrink-0" />
      Draft
    </span>
  )
}

function SkeletonRow() {
  return (
    <div className="flex items-center gap-6 px-6 py-4 border-b-2 border-ink/8 last:border-0 animate-pulse">
      <div className="flex-1 flex flex-col gap-1.5">
        <div className="h-4 w-48 bg-ink/10 rounded" />
        <div className="h-3 w-28 bg-ink/6 rounded" />
      </div>
      <div className="h-3 w-20 bg-ink/10 rounded" />
      <div className="h-3 w-24 bg-ink/10 rounded" />
    </div>
  )
}

export function PagesListPage() {
  const { data: pages, isLoading, isError } = usePages()

  return (
    <div className="flex flex-col gap-6">

      {/* Header */}
      <div>
        <h2 className="font-sans font-extrabold text-2xl tracking-tight text-ink">Pages</h2>
        {pages && (
          <p className="font-mono text-xs text-ink/50 mt-0.5">
            {pages.length} {pages.length === 1 ? 'page' : 'pages'}
          </p>
        )}
      </div>

      {/* Table */}
      <div
        className="border-2 border-ink rounded-xl overflow-hidden bg-white"
        style={{ boxShadow: 'var(--shadow-hard-xs)' }}
      >
        {/* Column headers */}
        <div className="grid grid-cols-[1fr_130px_120px] items-center px-6 py-3 bg-ink border-b-2 border-ink">
          <span className="font-mono text-[0.65rem] uppercase tracking-widest text-white/60">Page</span>
          <span className="font-mono text-[0.65rem] uppercase tracking-widest text-white/60">Status</span>
          <span className="font-mono text-[0.65rem] uppercase tracking-widest text-white/60">Updated</span>
        </div>

        {/* Loading */}
        {isLoading && (
          <>
            <SkeletonRow />
            <SkeletonRow />
            <SkeletonRow />
          </>
        )}

        {/* Error */}
        {isError && (
          <div className="flex items-center gap-3 px-6 py-8 text-brand-red">
            <AlertCircle size={16} className="shrink-0" />
            <p className="font-mono text-sm">Failed to load pages. Check your connection and try again.</p>
          </div>
        )}

        {/* Empty */}
        {!isLoading && !isError && pages?.length === 0 && (
          <div className="flex flex-col items-center gap-3 py-16 text-center">
            <FileText size={32} className="text-ink/20" />
            <p className="font-sans font-semibold text-ink/40">No pages yet</p>
            <p className="font-mono text-xs text-ink/30">
              Pages added to your site will appear here.
            </p>
          </div>
        )}

        {/* Rows */}
        {pages?.map((page) => (
          <Link
            key={page.id}
            to={`/pages/${page.slug}`}
            className="grid grid-cols-[1fr_130px_120px] items-center px-6 py-4 border-b-2 border-ink/8 last:border-0 hover:bg-ink/3 transition-colors group"
          >
            <div className="flex flex-col gap-0.5 min-w-0 pr-4">
              <span className="font-sans font-semibold text-sm text-ink group-hover:text-forest transition-colors truncate">
                {page.title}
              </span>
              <span className="font-mono text-xs text-ink/40 truncate">
                /{page.slug}
              </span>
            </div>

            <StatusBadge published={page.is_published} />

            <span className="font-mono text-xs text-ink/40">
              {formatDate(page.updated_at)}
            </span>
          </Link>
        ))}
      </div>
    </div>
  )
}
