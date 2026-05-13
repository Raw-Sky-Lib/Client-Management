import { useState } from 'react'
import { useNavigate } from 'react-router'
import { ChevronLeft } from 'lucide-react'
import { useTenantSupabase } from '@/contexts/supabase-context'
import { SaveIndicator, type SaveState } from '@/components/shared/save-indicator'
import { TiptapEditor } from './components/tiptap-editor'
import { PostMetaSidebar, type PostMeta } from './components/post-meta-sidebar'
import { slugify } from '@/lib/utils'
import { inputClass } from '@/features/pages/components/editor-primitives'

export function NewPostPage() {
  const navigate = useNavigate()
  const supabase = useTenantSupabase()

  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [meta, setMeta] = useState<PostMeta>({ slug: '', excerpt: '', author_name: '', cover_image_url: '' })
  const [saveState, setSaveState] = useState<SaveState>('idle')
  const [slugError, setSlugError] = useState('')

  function handleTitleChange(value: string) {
    setTitle(value)
    if (!meta.slug || meta.slug === slugify(title)) {
      setMeta(prev => ({ ...prev, slug: slugify(value) }))
    }
  }

  async function handleSlugBlur() {
    if (!meta.slug) return
    const { data } = await supabase
      .from('posts')
      .select('id')
      .eq('slug', meta.slug)
      .maybeSingle()
    setSlugError(data ? 'This slug is already taken.' : '')
  }

  async function handleSave() {
    if (!title.trim()) return
    setSaveState('saving')
    try {
      const { data, error } = await supabase
        .from('posts')
        .insert({
          title: title.trim(),
          slug: meta.slug || slugify(title),
          content,
          excerpt: meta.excerpt || null,
          author_name: meta.author_name || null,
          cover_image_url: meta.cover_image_url || null,
          is_published: false,
        })
        .select('id')
        .single()
      if (error) throw error
      navigate(`/blog/${data.id}/edit`, { replace: true })
    } catch {
      setSaveState('error')
      setTimeout(() => setSaveState('idle'), 3000)
    }
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
            New Post
          </h2>
        </div>

        <div className="flex items-center gap-3 shrink-0">
          <SaveIndicator state={saveState} />
          <button
            type="button"
            onClick={handleSave}
            disabled={!title.trim() || saveState === 'saving'}
            className="btn-ink-shadow flex items-center gap-2 bg-forest text-white font-sans font-extrabold text-sm uppercase tracking-wider py-3 px-6 rounded-lg border-2 border-ink disabled:opacity-50"
          >
            Save draft →
          </button>
        </div>
      </div>

      {/* Editor layout */}
      <div className="flex gap-6 items-start">
        {/* Main editor */}
        <div
          className="flex-1 min-w-0 border-2 border-ink rounded-xl overflow-hidden bg-white"
          style={{ boxShadow: 'var(--shadow-hard-xs)' }}
        >
          <div className="px-6 pt-6 pb-4 border-b-2 border-ink/10">
            <input
              className={`${inputClass} font-sans font-extrabold text-2xl py-2 border-0 border-b-2 border-ink/10 rounded-none px-0 focus:ring-0 focus:border-forest/40`}
              value={title}
              onChange={(e) => handleTitleChange(e.target.value)}
              placeholder="Post title…"
              autoFocus
            />
          </div>
          <TiptapEditor content={content} onChange={setContent} />
        </div>

        {/* Meta sidebar */}
        <PostMetaSidebar
          meta={meta}
          onChange={setMeta}
          onSlugBlur={handleSlugBlur}
          slugError={slugError}
        />
      </div>
    </div>
  )
}
