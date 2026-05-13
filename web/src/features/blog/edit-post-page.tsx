import { useState, useEffect, useRef, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router'
import { useQueryClient } from '@tanstack/react-query'
import { ChevronLeft, AlertCircle } from 'lucide-react'
import { useTenantSupabase } from '@/contexts/supabase-context'
import { SaveIndicator, type SaveState } from '@/components/shared/save-indicator'
import { TiptapEditor } from './components/tiptap-editor'
import { PostMetaSidebar, type PostMeta } from './components/post-meta-sidebar'
import { usePost } from './hooks/use-post'
import { slugify } from '@/lib/utils'
import { cn } from '@/lib/utils'
import { inputClass } from '@/features/pages/components/editor-primitives'
import api from '@/lib/axios'

export function EditPostPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const supabase = useTenantSupabase()
  const queryClient = useQueryClient()

  const { data: post, isLoading, isError } = usePost(id!)

  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [meta, setMeta] = useState<PostMeta>({ slug: '', excerpt: '', author_name: '', cover_image_url: '' })
  const [saveState, setSaveState] = useState<SaveState>('idle')
  const [isPublishing, setIsPublishing] = useState(false)
  const [slugError, setSlugError] = useState('')

  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const initialised = useRef(false)

  useEffect(() => {
    if (post && !initialised.current) {
      initialised.current = true
      setTitle(post.title)
      setContent(post.content ?? '')
      setMeta({
        slug: post.slug,
        excerpt: post.excerpt ?? '',
        author_name: post.author_name ?? '',
        cover_image_url: post.cover_image_url ?? '',
      })
    }
  }, [post])

  const save = useCallback(async (fields: {
    title: string
    content: string
    meta: PostMeta
  }) => {
    if (!id) return
    setSaveState('saving')
    try {
      const { error } = await supabase
        .from('posts')
        .update({
          title: fields.title,
          slug: fields.meta.slug,
          content: fields.content,
          excerpt: fields.meta.excerpt || null,
          author_name: fields.meta.author_name || null,
          cover_image_url: fields.meta.cover_image_url || null,
          updated_at: new Date().toISOString(),
        })
        .eq('id', id)
      if (error) throw error
      if (post?.slug) {
        api.post('/api/revalidate', { paths: [`/blog/${post.slug}`, '/blog'] }).catch(() => null)
      }
      queryClient.invalidateQueries({ queryKey: ['post', id] })
      queryClient.invalidateQueries({ queryKey: ['posts'] })
      setSaveState('saved')
      setTimeout(() => setSaveState('idle'), 2000)
    } catch {
      setSaveState('error')
      setTimeout(() => setSaveState('idle'), 3000)
    }
  }, [id, supabase, queryClient, post?.slug])

  function scheduleAutoSave(fields: { title: string; content: string; meta: PostMeta }) {
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => save(fields), 2000)
  }

  function handleTitleChange(value: string) {
    setTitle(value)
    const next = { title: value, content, meta }
    scheduleAutoSave(next)
  }

  function handleContentChange(html: string) {
    setContent(html)
    scheduleAutoSave({ title, content: html, meta })
  }

  function handleMetaChange(updated: PostMeta) {
    setMeta(updated)
    scheduleAutoSave({ title, content, meta: updated })
  }

  async function handleSlugBlur() {
    if (!meta.slug || meta.slug === post?.slug) {
      setSlugError('')
      return
    }
    const { data } = await supabase
      .from('posts')
      .select('id')
      .eq('slug', meta.slug)
      .neq('id', id!)
      .maybeSingle()
    setSlugError(data ? 'This slug is already taken.' : '')
  }

  async function handleTogglePublish() {
    if (!post || !id) return
    setIsPublishing(true)
    try {
      const isPublishing_ = !post.is_published
      const { error } = await supabase
        .from('posts')
        .update({
          is_published: isPublishing_,
          ...(isPublishing_ && !post.published_at
            ? { published_at: new Date().toISOString() }
            : {}),
        })
        .eq('id', id)
      if (!error) {
        api.post('/api/revalidate', { paths: [`/blog/${post.slug}`, '/blog'] }).catch(() => null)
        queryClient.invalidateQueries({ queryKey: ['post', id] })
        queryClient.invalidateQueries({ queryKey: ['posts'] })
      }
    } finally {
      setIsPublishing(false)
    }
  }

  if (isLoading) {
    return (
      <div className="flex flex-col gap-6 animate-pulse">
        <div className="h-8 w-64 bg-ink/10 rounded" />
        <div className="flex gap-6">
          <div className="flex-1 h-96 bg-ink/10 rounded-xl" />
          <div className="w-72 h-96 bg-ink/10 rounded-xl" />
        </div>
      </div>
    )
  }

  if (isError || !post) {
    return (
      <div className="flex items-center gap-3 text-brand-red">
        <AlertCircle size={16} />
        <p className="font-mono text-sm">Failed to load post. Check your connection and try again.</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Top bar */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3 min-w-0">
          <button
            type="button"
            onClick={() => navigate('/blog')}
            className="flex items-center gap-1 font-mono text-xs text-ink/50 hover:text-ink transition shrink-0"
          >
            <ChevronLeft size={14} />
            Blog
          </button>
          <span className="font-mono text-ink/25 shrink-0">/</span>
          <h2 className="font-sans font-extrabold text-xl tracking-tight text-ink truncate">
            {title || 'Untitled'}
          </h2>
        </div>

        <div className="flex items-center gap-3 shrink-0">
          <SaveIndicator state={saveState} />

          <span className={cn(
            'inline-flex items-center gap-1.5 font-mono text-xs font-bold uppercase tracking-widest',
            post.is_published ? 'text-forest' : 'text-ink/40',
          )}>
            <span className={cn(
              'w-1.5 h-1.5 rounded-full shrink-0',
              post.is_published ? 'bg-forest' : 'bg-ink/25',
            )} />
            {post.is_published ? 'Published' : 'Draft'}
          </span>

          <button
            type="button"
            onClick={handleTogglePublish}
            disabled={isPublishing}
            className="border-2 border-ink rounded-lg px-4 py-2 font-mono text-xs font-bold uppercase tracking-widest text-ink hover:bg-ink hover:text-cream transition disabled:opacity-50"
          >
            {isPublishing ? '…' : post.is_published ? 'Unpublish' : 'Publish'}
          </button>
        </div>
      </div>

      {/* Editor layout */}
      <div className="flex gap-6 items-start">
        {/* Main editor */}
        <div
          className="flex-1 min-w-0 border-2 border-ink rounded-xl overflow-hidden bg-white shadow-hard-xs"
        >
          <div className="px-6 pt-6 pb-4 border-b-2 border-ink/10">
            <input
              className={`${inputClass} font-sans font-extrabold text-2xl py-2 border-0 border-b-2 border-ink/10 rounded-none px-0 focus:ring-0 focus:border-forest/40`}
              value={title}
              onChange={(e) => handleTitleChange(e.target.value)}
              placeholder="Post title…"
            />
          </div>
          <TiptapEditor
            key={post.id}
            content={content}
            onChange={handleContentChange}
          />
        </div>

        {/* Meta sidebar */}
        <PostMetaSidebar
          meta={meta}
          onChange={handleMetaChange}
          onSlugBlur={handleSlugBlur}
          slugError={slugError}
        />
      </div>
    </div>
  )
}

// Re-export slug utility for meta auto-generation on new posts
export { slugify }
