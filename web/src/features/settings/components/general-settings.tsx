import { useState, useEffect } from 'react'
import { useUpdateSettings } from '../hooks/use-settings'
import { useTenantSupabase } from '@/contexts/supabase-context'
import { useQuery } from '@tanstack/react-query'
import { MediaPickerModal } from '@/features/media/components/media-picker-modal'
import { SaveIndicator, type SaveState } from '@/components/shared/save-indicator'
import { cn } from '@/lib/utils'

const KEYS = ['site_name', 'tagline', 'logo_url', 'contact_email'] as const

const labelClass = 'block font-mono text-[0.65rem] uppercase tracking-widest text-ink/50 mb-1.5'
const inputClass = [
  'w-full border-2 border-ink/15 rounded-lg px-3 py-2 font-mono text-xs text-ink',
  'placeholder:text-ink/25 focus:outline-none focus:border-ink transition bg-white',
].join(' ')

export function GeneralSettings() {
  const supabase = useTenantSupabase()
  const { mutateAsync: saveAll } = useUpdateSettings()
  const [saveState, setSaveState] = useState<SaveState>('idle')
  const [logoPickerOpen, setLogoPickerOpen] = useState(false)

  const { data: rows = [] } = useQuery({
    queryKey: ['settings', 'general-tab'],
    queryFn: async () => {
      const { data, error } = await supabase
        .from('site_settings')
        .select('key, value')
        .in('key', [...KEYS])
      if (error) throw error
      return data ?? []
    },
  })

  const [fields, setFields] = useState({ site_name: '', tagline: '', logo_url: '', contact_email: '' })

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

  return (
    <div className="flex flex-col gap-5">
      <div>
        <label className={labelClass}>Site name</label>
        <input type="text" value={fields.site_name} onChange={(e) => set('site_name', e.target.value)} placeholder="Acme Corp" className={inputClass} />
      </div>

      <div>
        <label className={labelClass}>Tagline</label>
        <input type="text" value={fields.tagline} onChange={(e) => set('tagline', e.target.value)} placeholder="Building the future" className={inputClass} />
      </div>

      <div>
        <label className={labelClass}>Logo URL</label>
        <div className="flex gap-2">
          <input type="text" value={fields.logo_url} onChange={(e) => set('logo_url', e.target.value)} placeholder="https://…" className={cn(inputClass, 'flex-1')} />
          <button
            type="button"
            onClick={() => setLogoPickerOpen(true)}
            className="border-2 border-ink/15 rounded-lg px-3 py-2 font-mono text-xs text-ink/50 hover:border-ink/40 hover:text-ink transition shrink-0"
          >
            Pick
          </button>
        </div>
        {fields.logo_url && (
          <img src={fields.logo_url} alt="Logo preview" className="mt-2 h-10 object-contain rounded border border-ink/10" />
        )}
      </div>

      <div>
        <label className={labelClass}>Contact email</label>
        <input type="email" value={fields.contact_email} onChange={(e) => set('contact_email', e.target.value)} placeholder="hello@example.com" className={inputClass} />
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
        open={logoPickerOpen}
        onClose={() => setLogoPickerOpen(false)}
        onSelect={(url) => { set('logo_url', url); setLogoPickerOpen(false) }}
      />
    </div>
  )
}
