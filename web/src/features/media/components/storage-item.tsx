import { useState } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { Folder, Copy, Check, Trash2, X, ZoomIn, Play } from 'lucide-react'
import { toast } from 'sonner'
import { cn, formatBytes } from '@/lib/utils'
import { useDeleteFile, getPublicUrl } from '../hooks/use-storage'
import { useTenantSupabase } from '@/contexts/supabase-context'
import type { StorageItem } from '../hooks/use-storage'

function isVideoMime(mime?: string) {
  return !!mime?.startsWith('video/')
}

interface StorageItemCardProps {
  item: StorageItem
  bucket: string
  folderPath: string
  selectable?: boolean
  onNavigate?: (folder: string) => void
  onSelect?: (url: string) => void
}

export function StorageItemCard({
  item,
  bucket,
  folderPath,
  selectable,
  onNavigate,
  onSelect,
}: StorageItemCardProps) {
  const supabase = useTenantSupabase()
  const { mutate: deleteFile, isPending: isDeleting } = useDeleteFile(bucket, folderPath)
  const [copied, setCopied] = useState(false)
  const [lightboxOpen, setLightboxOpen] = useState(false)

  const fullPath = folderPath ? `${folderPath}/${item.name}` : item.name
  const publicUrl = !item.isFolder ? getPublicUrl(supabase, bucket, fullPath) : ''
  const isVideo = isVideoMime((item.metadata as Record<string, string> | undefined)?.mimetype)

  function handleClick() {
    if (item.isFolder) {
      onNavigate?.(fullPath)
    } else if (selectable) {
      onSelect?.(publicUrl)
    } else {
      setLightboxOpen(true)
    }
  }

  async function handleCopy(e: React.MouseEvent) {
    e.stopPropagation()
    await navigator.clipboard.writeText(publicUrl)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  function handleDelete(e: React.MouseEvent) {
    e.stopPropagation()
    if (!confirm(`Delete "${item.name}"?`)) return
    deleteFile(fullPath, {
      onSuccess: () => toast.success(`Deleted "${item.name}"`),
      onError: () => toast.error(`Failed to delete "${item.name}"`),
    })
  }

  return (
    <>
      <div
        className={cn(
          'group border-2 rounded-xl overflow-hidden bg-white transition cursor-pointer',
          item.isFolder
            ? 'border-ink/15 hover:border-amber-400/60'
            : selectable
              ? 'border-ink/15 hover:border-forest'
              : 'border-ink/15 hover:border-ink/40',
          isDeleting && 'opacity-50 pointer-events-none',
        )}
        onClick={handleClick}
      >
        {/* Thumbnail / folder icon */}
        <div className={cn(
          'aspect-video flex items-center justify-center overflow-hidden relative',
          item.isFolder ? 'bg-amber-50' : 'bg-ink/4',
        )}>
          {item.isFolder ? (
            <div className="flex flex-col items-center gap-1.5">
              <Folder
                size={40}
                className="text-amber-400 group-hover:text-amber-500 transition fill-amber-100 group-hover:fill-amber-200"
              />
              <span className="font-mono text-[0.6rem] text-amber-500/70 uppercase tracking-widest">Folder</span>
            </div>
          ) : isVideo ? (
            <>
              <video
                src={publicUrl}
                preload="metadata"
                muted
                playsInline
                className="w-full h-full object-cover"
              />
              <div className="absolute inset-0 bg-ink/20 group-hover:bg-ink/40 transition flex items-center justify-center">
                <div className="w-8 h-8 rounded-full bg-white/90 flex items-center justify-center shadow">
                  <Play size={14} className="text-ink fill-ink ml-0.5" />
                </div>
              </div>
            </>
          ) : (
            <>
              <img
                src={publicUrl}
                alt={item.name}
                className="w-full h-full object-cover"
                loading="lazy"
              />
              {!selectable && (
                <div className="absolute inset-0 bg-ink/0 group-hover:bg-ink/20 transition flex items-center justify-center">
                  <ZoomIn size={20} className="text-white opacity-0 group-hover:opacity-100 transition drop-shadow" />
                </div>
              )}
            </>
          )}
        </div>

        {/* Info + actions */}
        <div className="px-3 py-2.5 flex items-center gap-2">
          <div className="flex-1 min-w-0">
            <p className="font-mono text-xs text-ink truncate" title={item.name}>
              {item.name}
            </p>
            {!item.isFolder && item.metadata && (
              <p className="font-mono text-[0.65rem] text-ink/40 mt-0.5">
                {formatBytes(item.metadata.size)}
              </p>
            )}
          </div>

          {!item.isFolder && !selectable && (
            <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition shrink-0">
              <button
                type="button"
                onClick={handleCopy}
                className="p-1.5 rounded hover:bg-ink/5 transition text-ink/40 hover:text-ink"
                aria-label="Copy URL"
              >
                {copied ? <Check size={13} className="text-forest" /> : <Copy size={13} />}
              </button>
              <button
                type="button"
                onClick={handleDelete}
                className="p-1.5 rounded hover:bg-brand-red/10 transition text-ink/40 hover:text-brand-red"
                aria-label="Delete"
              >
                <Trash2 size={13} />
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Lightbox */}
      {!item.isFolder && (
        <Dialog.Root open={lightboxOpen} onOpenChange={setLightboxOpen}>
          <Dialog.Portal>
            <Dialog.Overlay className="fixed inset-0 z-50 bg-ink/80 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />
            <Dialog.Content
              className="fixed inset-0 z-50 flex flex-col items-center justify-center p-4 outline-none"
              onClick={() => setLightboxOpen(false)}
            >
              {/* Filename + actions bar */}
              <div
                className="flex items-center justify-between w-full max-w-5xl mb-3"
                onClick={(e) => e.stopPropagation()}
              >
                <p className="font-mono text-xs text-white/70 truncate">{item.name}</p>
                <div className="flex items-center gap-2 shrink-0">
                  <button
                    type="button"
                    onClick={handleCopy}
                    className="flex items-center gap-1.5 font-mono text-xs text-white/60 hover:text-white transition px-2 py-1 rounded hover:bg-white/10"
                  >
                    {copied ? <Check size={12} className="text-forest" /> : <Copy size={12} />}
                    {copied ? 'Copied' : 'Copy URL'}
                  </button>
                  <Dialog.Close asChild>
                    <button
                      type="button"
                      className="p-1.5 rounded hover:bg-white/10 transition text-white/60 hover:text-white"
                      aria-label="Close"
                    >
                      <X size={16} />
                    </button>
                  </Dialog.Close>
                </div>
              </div>

              {/* Media */}
              <div
                className="max-w-5xl max-h-[80vh] w-full flex items-center justify-center"
                onClick={(e) => e.stopPropagation()}
              >
                {isVideo ? (
                  <video
                    src={publicUrl}
                    controls
                    autoPlay
                    className="max-w-full max-h-[80vh] rounded-xl shadow-2xl"
                  />
                ) : (
                  <img
                    src={publicUrl}
                    alt={item.name}
                    className="max-w-full max-h-[80vh] object-contain rounded-xl shadow-2xl"
                  />
                )}
              </div>

              {item.metadata && (
                <p
                  className="font-mono text-[0.65rem] text-white/30 mt-3"
                  onClick={(e) => e.stopPropagation()}
                >
                  {formatBytes(item.metadata.size)}
                </p>
              )}

              <Dialog.Title className="sr-only">{item.name}</Dialog.Title>
            </Dialog.Content>
          </Dialog.Portal>
        </Dialog.Root>
      )}
    </>
  )
}
