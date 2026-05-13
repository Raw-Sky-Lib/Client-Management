import { ImageIcon } from 'lucide-react'
import { Field, inputClass, textareaClass } from '@/features/pages/components/editor-primitives'

export interface PostMeta {
  slug: string
  excerpt: string
  author_name: string
  cover_image_url: string
}

interface PostMetaSidebarProps {
  meta: PostMeta
  onChange: (meta: PostMeta) => void
  onSlugBlur?: () => void
  slugError?: string
}

export function PostMetaSidebar({ meta, onChange, onSlugBlur, slugError }: PostMetaSidebarProps) {
  function set<K extends keyof PostMeta>(key: K, value: PostMeta[K]) {
    onChange({ ...meta, [key]: value })
  }

  return (
    <div
      className="w-72 shrink-0 border-2 border-ink rounded-xl overflow-hidden bg-white sticky top-0"
      style={{ boxShadow: 'var(--shadow-hard-xs)' }}
    >
      <div className="px-4 py-3 bg-ink border-b-2 border-ink">
        <span className="font-mono text-[0.65rem] uppercase tracking-widest text-white/60">
          Post Settings
        </span>
      </div>

      <div className="p-4 flex flex-col gap-5">
        <Field label="Slug">
          <input
            className={inputClass}
            value={meta.slug}
            onChange={(e) => set('slug', e.target.value)}
            onBlur={onSlugBlur}
            placeholder="post-url-slug"
          />
          {slugError && (
            <p className="font-mono text-xs text-brand-red">{slugError}</p>
          )}
        </Field>

        <Field label="Excerpt">
          <textarea
            className={textareaClass}
            rows={3}
            value={meta.excerpt}
            onChange={(e) => set('excerpt', e.target.value)}
            placeholder="Short description shown in blog listings…"
          />
        </Field>

        <Field label="Author">
          <input
            className={inputClass}
            value={meta.author_name}
            onChange={(e) => set('author_name', e.target.value)}
            placeholder="Author name"
          />
        </Field>

        <Field label="Cover Image">
          {meta.cover_image_url ? (
            <div className="flex flex-col gap-2">
              <img
                src={meta.cover_image_url}
                alt="Cover"
                className="w-full aspect-video object-cover rounded-lg border-2 border-ink/10"
              />
              <button
                type="button"
                onClick={() => set('cover_image_url', '')}
                className="font-mono text-xs text-ink/40 hover:text-brand-red transition text-left"
              >
                Remove image
              </button>
            </div>
          ) : (
            <button
              type="button"
              disabled
              className="w-full aspect-video flex flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed border-ink/20 bg-ink/3 text-ink/30 cursor-not-allowed"
              title="Media picker coming soon"
            >
              <ImageIcon size={20} />
              <span className="font-mono text-xs">Choose image</span>
            </button>
          )}
        </Field>
      </div>
    </div>
  )
}
