import { Link } from 'react-router'
import { BookOpen, AlertCircle } from 'lucide-react'
import { formatDate } from '@/lib/utils'
import type { Post } from '@/types'

type PostListItem = Pick<Post, 'id' | 'slug' | 'title' | 'is_published' | 'published_at' | 'updated_at'>

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
    <div className="grid grid-cols-[1fr_120px_130px_100px] items-center px-6 py-4 border-b-2 border-ink/8 last:border-0 animate-pulse">
      <div className="flex flex-col gap-1.5">
        <div className="h-4 w-56 bg-ink/10 rounded" />
        <div className="h-3 w-32 bg-ink/6 rounded" />
      </div>
      <div className="h-3 w-20 bg-ink/10 rounded" />
      <div className="h-3 w-24 bg-ink/10 rounded" />
      <div className="h-3 w-20 bg-ink/10 rounded" />
    </div>
  )
}

interface PostsTableProps {
  posts?: PostListItem[]
  isLoading: boolean
  isError: boolean
}

export function PostsTable({ posts, isLoading, isError }: PostsTableProps) {
  return (
    <div
      className="border-2 border-ink rounded-xl overflow-hidden bg-white"
      style={{ boxShadow: 'var(--shadow-hard-xs)' }}
    >
      <div className="grid grid-cols-[1fr_120px_130px_100px] items-center px-6 py-3 bg-ink border-b-2 border-ink">
        <span className="font-mono text-[0.65rem] uppercase tracking-widest text-white/60">Post</span>
        <span className="font-mono text-[0.65rem] uppercase tracking-widest text-white/60">Status</span>
        <span className="font-mono text-[0.65rem] uppercase tracking-widest text-white/60">Published</span>
        <span className="font-mono text-[0.65rem] uppercase tracking-widest text-white/60">Updated</span>
      </div>

      {isLoading && (
        <>
          <SkeletonRow />
          <SkeletonRow />
          <SkeletonRow />
        </>
      )}

      {isError && (
        <div className="flex items-center gap-3 px-6 py-8 text-brand-red">
          <AlertCircle size={16} className="shrink-0" />
          <p className="font-mono text-sm">Failed to load posts. Check your connection and try again.</p>
        </div>
      )}

      {!isLoading && !isError && posts?.length === 0 && (
        <div className="flex flex-col items-center gap-3 py-16 text-center">
          <BookOpen size={32} className="text-ink/20" />
          <p className="font-sans font-semibold text-ink/40">No posts yet</p>
          <p className="font-mono text-xs text-ink/30">Create your first post to get started.</p>
        </div>
      )}

      {posts?.map((post) => (
        <Link
          key={post.id}
          to={`/blog/${post.id}/edit`}
          className="grid grid-cols-[1fr_120px_130px_100px] items-center px-6 py-4 border-b-2 border-ink/8 last:border-0 hover:bg-ink/3 transition-colors group"
        >
          <div className="flex flex-col gap-0.5 min-w-0 pr-4">
            <span className="font-sans font-semibold text-sm text-ink group-hover:text-forest transition-colors truncate">
              {post.title || <span className="italic text-ink/30">Untitled</span>}
            </span>
            <span className="font-mono text-xs text-ink/40 truncate">/{post.slug}</span>
          </div>
          <StatusBadge published={post.is_published} />
          <span className="font-mono text-xs text-ink/40">
            {post.published_at ? formatDate(post.published_at) : <span className="text-ink/25">—</span>}
          </span>
          <span className="font-mono text-xs text-ink/40">{formatDate(post.updated_at)}</span>
        </Link>
      ))}
    </div>
  )
}
