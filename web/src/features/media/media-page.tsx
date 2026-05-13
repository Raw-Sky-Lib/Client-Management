import { AlertCircle } from 'lucide-react'
import { useInitBucket } from './hooks/use-storage'
import { FileBrowser } from './components/file-browser'

export function MediaPage() {
  const { data: init, isLoading, isError } = useInitBucket()

  if (isLoading) {
    return (
      <div className="flex flex-col gap-6 animate-pulse">
        <div className="h-8 w-32 bg-ink/10 rounded" />
        <div className="h-3 w-24 bg-ink/6 rounded" />
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="border-2 border-ink/10 rounded-xl overflow-hidden">
              <div className="aspect-video bg-ink/10" />
              <div className="px-3 py-2.5 flex flex-col gap-1.5">
                <div className="h-3 w-20 bg-ink/10 rounded" />
              </div>
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (isError || !init?.bucket) {
    return (
      <div className="flex items-center gap-3 text-brand-red">
        <AlertCircle size={16} />
        <p className="font-mono text-sm">Failed to initialise storage. Check your connection and try again.</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h2 className="font-sans font-extrabold text-2xl tracking-tight text-ink">Media</h2>
        <p className="font-mono text-xs text-ink/50 mt-0.5">{init.bucket}</p>
      </div>
      <FileBrowser bucket={init.bucket} />
    </div>
  )
}
