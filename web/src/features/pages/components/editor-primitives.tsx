import type { ReactNode } from 'react'

export const inputClass =
  'w-full border-2 border-ink rounded-lg px-4 py-3 font-sans text-base text-ink bg-white ' +
  'placeholder:text-ink/30 outline-none focus:ring-2 focus:ring-forest/20 focus:border-forest transition'

export const textareaClass =
  inputClass + ' resize-none leading-relaxed'

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <label className="font-mono text-xs uppercase tracking-widest text-ink/60">
        {label}
      </label>
      {children}
    </div>
  )
}
