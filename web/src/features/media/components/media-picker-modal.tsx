import * as Dialog from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import { useInitBucket } from '../hooks/use-storage'
import { FileBrowser } from './file-browser'

interface MediaPickerModalProps {
  open: boolean
  onClose: () => void
  onSelect: (url: string) => void
}

export function MediaPickerModal({ open, onClose, onSelect }: MediaPickerModalProps) {
  const { data: init, isLoading } = useInitBucket()

  function handleSelect(url: string) {
    onSelect(url)
    onClose()
  }

  return (
    <Dialog.Root open={open} onOpenChange={(v) => !v && onClose()}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-ink/40 z-40 animate-in fade-in duration-150" />
        <Dialog.Content className="fixed inset-4 md:inset-8 lg:inset-16 z-50 flex flex-col bg-cream border-2 border-ink rounded-2xl overflow-hidden shadow-hard">

          <div className="flex items-center justify-between px-6 py-4 bg-ink border-b-2 border-ink shrink-0">
            <Dialog.Title className="font-sans font-extrabold text-white">
              Choose Image
            </Dialog.Title>
            <button
              type="button"
              onClick={onClose}
              aria-label="Close"
              className="text-white/60 hover:text-white transition"
            >
              <X size={18} />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-6">
            {isLoading || !init?.bucket ? (
              <div className="flex items-center justify-center py-20">
                <p className="font-mono text-xs text-ink/40">Initialising storage…</p>
              </div>
            ) : (
              <FileBrowser bucket={init.bucket} selectable onSelect={handleSelect} />
            )}
          </div>

        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
