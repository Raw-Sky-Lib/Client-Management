import type { ReactNode } from 'react'

export const inputClass =
  'w-full border-2 border-ink/20 rounded-lg px-3 py-2 font-mono text-xs text-ink bg-white ' +
  'placeholder:text-ink/25 outline-none focus:border-ink transition'

export const textareaClass = inputClass + ' resize-none leading-relaxed'

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <label className="font-mono text-[0.6rem] uppercase tracking-widest text-ink/40">
        {label}
      </label>
      {children}
    </div>
  )
}
