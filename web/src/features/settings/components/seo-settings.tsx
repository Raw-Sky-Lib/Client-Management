import { useState, useEffect } from 'react'
import { useUpdateSettings } from '../hooks/use-settings'
import { useTenantSupabase } from '@/contexts/supabase-context'
import { useQuery } from '@tanstack/react-query'
import { MediaPickerModal } from '@/features/media/components/media-picker-modal'
import { SaveIndicator, type SaveState } from '@/components/shared/save-indicator'
import { cn } from '@/lib/utils'

const KEYS = ['seo_title', 'seo_description', 'og_image_url'] as const

const labelClass = 'block font-mono text-[0.65rem] uppercase tracking-widest text-ink/50 mb-1.5'
const inputClass = [
  'w-full border-2 border-ink/15 rounded-lg px-3 py-2 font-mono text-xs text-ink',
  'placeholder:text-ink/25 focus:outline-none focus:border-ink transition bg-white',
].join(' ')

export function SeoSettings() {
  const supabase = useTenantSupabase()
  const { mutateAsync: saveAll } = useUpdateSettings()
  const [saveState, setSaveState] = useState<SaveState>('idle')
  const [ogPickerOpen, setOgPickerOpen] = useState(false)

  const { data: rows = [] } = useQuery({
    queryKey: ['settings', 'seo-tab'],
    queryFn: async () => {
      const { data, error } = await supabase
        .from('site_settings')
        .select('key, value')
        .in('key', [...KEYS])
      if (error) throw error
      return data ?? []
    },
  })

  const [fields, setFields] = useState({ seo_title: '', seo_description: '', og_image_url: '' })

  useEffect(() => {
    const map = Object.fromEntries(rows.map((r: { key: string; value: string }) => [r.key, r.value]))
    setFields((prev) => ({ ...prev, ...map }))
  }, [rows])

  function set(key: keyof typeof fields, value: string) {
    setFields((prev) => ({ ...prev, [key]: value }))
  }

  async function handleSave() {
    setSaveState('saving')
    try {
      await saveAll(Object.entries(fields).map(([key, value]) => ({ key, value })))
      setSaveState('saved')
      setTimeout(() => setSaveState('idle'), 2000)
    } catch {
      setSaveState('error')
      setTimeout(() => setSaveState('idle'), 3000)
    }
  }

  const descLen = fields.seo_description.length

  return (
    <div className="flex flex-col gap-5">
      <div>
        <label className={labelClass}>Default SEO title</label>
        <input
          type="text"
          value={fields.seo_title}
          onChange={(e) => set('seo_title', e.target.value)}
          placeholder="Acme Corp — Building the Future"
          className={inputClass}
        />
        <p className="mt-1 font-mono text-[0.6rem] text-ink/35">Appears in browser tab and search results when no page-level title is set.</p>
      </div>

      <div>
        <div className="flex items-center justify-between mb-1.5">
          <label className={labelClass} style={{ marginBottom: 0 }}>Meta description</label>
          <span className={cn('font-mono text-[0.6rem]', descLen > 160 ? 'text-red-500' : 'text-ink/35')}>
            {descLen}/160
          </span>
        </div>
        <textarea
          value={fields.seo_description}
          onChange={(e) => set('seo_description', e.target.value)}
          placeholder="A short description of your site for search engines…"
          rows={3}
          className={cn(inputClass, 'resize-none')}
        />
      </div>

      <div>
        <label className={labelClass}>OG image URL</label>
        <div className="flex gap-2">
          <input
            type="text"
            value={fields.og_image_url}
            onChange={(e) => set('og_image_url', e.target.value)}
            placeholder="https://…"
            className={cn(inputClass, 'flex-1')}
          />
          <button
            type="button"
            onClick={() => setOgPickerOpen(true)}
            className="border-2 border-ink/15 rounded-lg px-3 py-2 font-mono text-xs text-ink/50 hover:border-ink/40 hover:text-ink transition shrink-0"
          >
            Pick
          </button>
        </div>
        <p className="mt-1 font-mono text-[0.6rem] text-ink/35">Shown when links are shared on social media. Recommended: 1200×630px.</p>
        {fields.og_image_url && (
          <img src={fields.og_image_url} alt="OG image preview" className="mt-2 h-16 w-full object-cover rounded border border-ink/10" />
        )}
      </div>

      <div className="flex items-center justify-between pt-2 border-t-2 border-ink/8">
        <SaveIndicator state={saveState} />
        <button
          type="button"
          onClick={handleSave}
          disabled={saveState === 'saving'}
          className="border-2 border-ink rounded-xl px-5 py-2.5 font-mono text-xs font-bold uppercase tracking-widest text-ink hover:bg-ink hover:text-cream transition disabled:opacity-40"
        >
          Save
        </button>
      </div>

      <MediaPickerModal
        open={ogPickerOpen}
        onClose={() => setOgPickerOpen(false)}
        onSelect={(url) => { set('og_image_url', url); setOgPickerOpen(false) }}
      />
    </div>
  )
}
