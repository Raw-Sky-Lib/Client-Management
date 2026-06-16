import { useState } from 'react'
import { Code, Eye, EyeOff, X } from 'lucide-react'
import { cn } from '@/lib/utils'

interface EmbedCodeFieldProps {
  label: string
  value: string
  onChange: (code: string) => void
  placeholder?: string
}

export function EmbedCodeField({ label, value, onChange, placeholder }: EmbedCodeFieldProps) {
  const [previewOpen, setPreviewOpen] = useState(false)

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="font-mono text-[0.6rem] uppercase tracking-widest text-ink/50">{label}</span>
        {value && (
          <button
            type="button"
            onClick={() => setPreviewOpen(v => !v)}
            className="flex items-center gap-1 font-mono text-[0.6rem] uppercase tracking-widest text-ink/40 hover:text-ink transition"
          >
            {previewOpen ? <EyeOff size={10} /> : <Eye size={10} />}
            {previewOpen ? 'Hide' : 'Preview'}
          </button>
        )}
      </div>

      <div className={cn(
        'relative border-2 rounded-lg overflow-hidden transition-colors',
        value ? 'border-ink/20' : 'border-ink/10',
      )}>
        <div className="flex items-start gap-2 p-2.5">
          <Code size={13} className="text-ink/30 mt-0.5 shrink-0" />
          <textarea
            rows={4}
            value={value}
            onChange={e => onChange(e.target.value)}
            placeholder={placeholder ?? '<iframe src="https://maps.google.com/..." ...></iframe>'}
            spellCheck={false}
            className="flex-1 font-mono text-[0.7rem] text-ink/80 bg-transparent outline-none resize-none placeholder:text-ink/25 leading-relaxed"
          />
          {value && (
            <button
              type="button"
              onClick={() => onChange('')}
              className="text-ink/25 hover:text-brand-red transition mt-0.5 shrink-0"
              aria-label="Clear embed code"
            >
              <X size={12} />
            </button>
          )}
        </div>
      </div>

      {value && previewOpen && (
        <div className="border-2 border-ink/10 rounded-lg overflow-hidden bg-white">
          <div className="px-3 py-1.5 border-b border-ink/8 flex items-center gap-1.5">
            <span className="w-2 h-2 rounded-full bg-ink/10" />
            <span className="font-mono text-[0.55rem] text-ink/30 uppercase tracking-widest">Preview</span>
          </div>
          <iframe
            title="embed-preview"
            srcDoc={value}
            sandbox="allow-scripts allow-popups allow-forms allow-presentation"
            className="w-full h-75 border-0"
          />
        </div>
      )}
    </div>
  )
}
