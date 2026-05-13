import { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Plus, Trash2 } from 'lucide-react'
import { useTenantSupabase } from '@/contexts/supabase-context'
import { useUpdateSetting } from '../hooks/use-settings'
import { SaveIndicator, type SaveState } from '@/components/shared/save-indicator'

interface SocialLink {
  _key: string
  label: string
  url: string
}

const inputClass = [
  'border-2 border-ink/15 rounded-lg px-3 py-2 font-mono text-xs text-ink',
  'placeholder:text-ink/25 focus:outline-none focus:border-ink transition bg-white',
].join(' ')

const SUGGESTIONS = ['Instagram', 'X / Twitter', 'Facebook', 'LinkedIn', 'YouTube', 'TikTok', 'Pinterest', 'GitHub', 'Behance', 'Dribbble']

let _keyCounter = 0
function genKey() { return `sl-${++_keyCounter}` }

export function SocialSettings() {
  const supabase = useTenantSupabase()
  const { mutateAsync: saveSetting } = useUpdateSetting()
  const [saveState, setSaveState] = useState<SaveState>('idle')
  const [links, setLinks] = useState<SocialLink[]>([])

  const { data: raw = '' } = useQuery({
    queryKey: ['settings', 'social_links'],
    queryFn: async () => {
      const { data, error } = await supabase
        .from('site_settings')
        .select('value')
        .eq('key', 'social_links')
        .maybeSingle()
      if (error) throw error
      return data?.value ?? ''
    },
  })

  useEffect(() => {
    try {
      const parsed: { label: string; url: string }[] = raw ? JSON.parse(raw) : []
      setLinks(parsed.map((l) => ({ ...l, _key: genKey() })))
    } catch {
      setLinks([])
    }
  }, [raw])

  function addLink() {
    setLinks((prev) => [...prev, { _key: genKey(), label: '', url: '' }])
  }

  function removeLink(key: string) {
    setLinks((prev) => prev.filter((l) => l._key !== key))
  }

  function updateLink(key: string, field: 'label' | 'url', value: string) {
    setLinks((prev) => prev.map((l) => (l._key === key ? { ...l, [field]: value } : l)))
  }

  async function handleSave() {
    setSaveState('saving')
    try {
      const payload = links.map(({ label, url }) => ({ label, url }))
      await saveSetting({ key: 'social_links', value: JSON.stringify(payload) })
      setSaveState('saved')
      setTimeout(() => setSaveState('idle'), 2000)
    } catch {
      setSaveState('error')
      setTimeout(() => setSaveState('idle'), 3000)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2 pb-2 border-b-2 border-ink/8">
        <span className="font-mono text-[0.6rem] uppercase tracking-widest text-ink/35 w-28">Platform</span>
        <span className="font-mono text-[0.6rem] uppercase tracking-widest text-ink/35 flex-1">URL</span>
        <span className="w-5.25" />
      </div>

      <div className="flex flex-col gap-2">
        {links.map((link) => (
          <div key={link._key} className="flex items-center gap-2">
            <input
              type="text"
              value={link.label}
              onChange={(e) => updateLink(link._key, 'label', e.target.value)}
              placeholder="Platform"
              list="social-suggestions"
              className={`${inputClass} w-28`}
            />
            <input
              type="url"
              value={link.url}
              onChange={(e) => updateLink(link._key, 'url', e.target.value)}
              placeholder="https://…"
              className={`${inputClass} flex-1`}
            />
            <button
              type="button"
              aria-label="Remove link"
              onClick={() => removeLink(link._key)}
              className="text-ink/25 hover:text-red-500 transition shrink-0"
            >
              <Trash2 size={13} />
            </button>
          </div>
        ))}
      </div>

      <datalist id="social-suggestions">
        {SUGGESTIONS.map((s) => <option key={s} value={s} />)}
      </datalist>

      {links.length === 0 && (
        <p className="font-mono text-[0.65rem] text-ink/30 text-center py-3">No social links yet.</p>
      )}

      <button
        type="button"
        onClick={addLink}
        className="flex items-center gap-1.5 font-mono text-xs text-ink/50 hover:text-ink transition self-start"
      >
        <Plus size={13} />
        Add link
      </button>

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
    </div>
  )
}
