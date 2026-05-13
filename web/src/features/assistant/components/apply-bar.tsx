import { cn } from '@/lib/utils'
import { SaveIndicator, type SaveState } from '@/components/shared/save-indicator'

interface ApplyBarProps {
  selectedCount: number
  saveState: SaveState
  onApply: () => void
  onDiscard: () => void
}

export function ApplyBar({ selectedCount, saveState, onApply, onDiscard }: ApplyBarProps) {
  const isApplying = saveState === 'saving'

  return (
    <div className="px-4 py-3 border-t-2 border-ink/10 flex items-center justify-between gap-3 bg-white">
      <SaveIndicator state={saveState} />

      <div className="flex items-center gap-2 ml-auto">
        <button
          type="button"
          onClick={onDiscard}
          disabled={isApplying}
          className="font-mono text-xs text-ink/50 hover:text-ink transition disabled:opacity-40 px-3 py-2"
        >
          Discard
        </button>
        <button
          type="button"
          onClick={onApply}
          disabled={selectedCount === 0 || isApplying}
          className={cn(
            'flex items-center gap-2 border-2 border-ink rounded-xl px-5 py-2.5',
            'font-mono text-xs font-bold uppercase tracking-widest transition',
            selectedCount > 0 && !isApplying
              ? 'bg-forest text-white border-forest hover:bg-forest-deep'
              : 'opacity-40 cursor-not-allowed text-ink',
          )}
        >
          {isApplying ? 'Applying…' : `Apply ${selectedCount} change${selectedCount !== 1 ? 's' : ''} →`}
        </button>
      </div>
    </div>
  )
}
