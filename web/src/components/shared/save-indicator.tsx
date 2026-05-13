import { cn } from '@/lib/utils'

export type SaveState = 'idle' | 'saving' | 'saved' | 'error'

export function SaveIndicator({ state }: { state: SaveState }) {
  if (state === 'idle') return null

  return (
    <span className={cn(
      'font-mono text-xs font-bold',
      state === 'saving' && 'text-ink/40',
      state === 'saved'  && 'text-forest',
      state === 'error'  && 'text-brand-red',
    )}>
      {state === 'saving' && '● Saving…'}
      {state === 'saved'  && '✓ Saved'}
      {state === 'error'  && '✕ Failed to save'}
    </span>
  )
}
