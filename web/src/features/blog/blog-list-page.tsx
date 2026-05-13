import { Link } from 'react-router'
import { usePosts } from './hooks/use-posts'
import { PostsTable } from './components/posts-table'

export function BlogListPage() {
  const { data: posts, isLoading, isError } = usePosts()

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="font-sans font-extrabold text-2xl tracking-tight text-ink">Blog</h2>
          {posts && (
            <p className="font-mono text-xs text-ink/50 mt-0.5">
              {posts.length} {posts.length === 1 ? 'post' : 'posts'}
            </p>
          )}
        </div>
        <Link
          to="/blog/new"
          className="border-2 border-ink rounded-lg px-4 py-2 font-mono text-xs font-bold uppercase tracking-widest text-ink hover:bg-ink hover:text-cream transition"
        >
          New post →
        </Link>
      </div>

      <PostsTable posts={posts} isLoading={isLoading} isError={isError} />
    </div>
  )
}
