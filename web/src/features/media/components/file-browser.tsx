import { useState, useRef, useEffect, useCallback } from 'react'
import {
  ChevronRight, Home, AlertCircle, ImageIcon,
  FolderPlus, ChevronDown, Check, AlertTriangle, X,
} from 'lucide-react'
import { toast } from 'sonner'
import { useStorageFiles, useUploadFile, validateFile } from '../hooks/use-storage'
import { StorageItemCard } from './storage-item'
import { cn } from '@/lib/utils'

interface FileBrowserProps {
  bucket: string
  selectable?: boolean
  onSelect?: (url: string) => void
}

type FailedFile = { name: string; reason: string }
type DupeWarning = { files: File[]; dupes: string[] }

function SkeletonCard() {
  return (
    <div className="border-2 border-ink/10 rounded-xl overflow-hidden bg-white animate-pulse">
      <div className="aspect-video bg-ink/10" />
      <div className="px-3 py-2.5 flex flex-col gap-1.5">
        <div className="h-3 w-24 bg-ink/10 rounded" />
        <div className="h-2.5 w-12 bg-ink/6 rounded" />
      </div>
    </div>
  )
}

export function FileBrowser({ bucket, selectable, onSelect }: FileBrowserProps) {
  const [pathStack, setPathStack] = useState<string[]>([])
  const [isDragging, setIsDragging] = useState(false)
  const [uploadPercent, setUploadPercent] = useState<number | null>(null)
  const [failedFiles, setFailedFiles] = useState<FailedFile[]>([])
  const [dupeWarning, setDupeWarning] = useState<DupeWarning | null>(null)
  const progressMap = useRef<Map<string, number>>(new Map())
  const progressFillRef = useRef<HTMLDivElement>(null)

  // Folder picker state
  const [pickerOpen, setPickerOpen] = useState(false)
  const [uploadDestFolder, setUploadDestFolder] = useState('')
  const [newFolderInput, setNewFolderInput] = useState('')
  const pickerRef = useRef<HTMLDivElement>(null)

  const currentPath = pathStack.join('/')
  const { data: items, isLoading, isError } = useStorageFiles(bucket, currentPath)
  const { mutateAsync: upload, isPending: isUploading } = useUploadFile(bucket)

  const folders = (items ?? []).filter((i) => i.isFolder)

  // Drive the progress bar fill via DOM — avoids JSX inline style for a dynamic value
  useEffect(() => {
    progressFillRef.current?.style.setProperty(
      '--progress',
      uploadPercent !== null ? `${uploadPercent}%` : '0%',
    )
  }, [uploadPercent])

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (pickerRef.current && !pickerRef.current.contains(e.target as Node)) {
        setPickerOpen(false)
      }
    }
    if (pickerOpen) document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [pickerOpen])

  function navigateInto(folder: string) {
    setPathStack(folder.split('/'))
    setUploadDestFolder('')
    setDupeWarning(null)
    setFailedFiles([])
  }

  function navigateTo(index: number) {
    setPathStack((prev) => prev.slice(0, index))
    setUploadDestFolder('')
    setDupeWarning(null)
    setFailedFiles([])
  }

  const effectiveUploadPath = uploadDestFolder
    ? (currentPath ? `${currentPath}/${uploadDestFolder}` : uploadDestFolder)
    : currentPath

  const destLabel = uploadDestFolder
    ? uploadDestFolder
    : currentPath ? pathStack.at(-1)! : 'root'

  // Core upload — runs after all checks pass
  const doUpload = useCallback(async (filesToUpload: File[]) => {
    setFailedFiles([])
    // Don't pre-set progress to 0 — only show the bar when real bytes start moving.
    // This avoids a flash for files that fail immediately (validation, network refusal, etc).
    progressMap.current = new Map(filesToUpload.map((f) => [f.name, 0]))

    const results = await Promise.allSettled(
      filesToUpload.map((file) =>
        upload({
          file,
          folderPath: effectiveUploadPath,
          onProgress: (pct) => {
            progressMap.current.set(file.name, pct)
            const vals = [...progressMap.current.values()]
            setUploadPercent(Math.round(vals.reduce((a, b) => a + b, 0) / vals.length))
          },
        }),
      ),
    )

    setUploadPercent(null)
    progressMap.current.clear()

    const failed: FailedFile[] = results
      .map((r, i) => ({ r, name: filesToUpload[i].name }))
      .filter(({ r }) => r.status === 'rejected')
      .map(({ r, name }) => ({
        name,
        reason: r.status === 'rejected'
          ? (r.reason instanceof Error ? r.reason.message : 'Upload failed.')
          : '',
      }))

    const successCount = results.filter((r) => r.status === 'fulfilled').length

    if (failed.length === 0) {
      toast.success(
        successCount === 1
          ? `Uploaded "${filesToUpload[0].name}"`
          : `Uploaded ${successCount} files`,
      )
    } else if (successCount > 0) {
      toast.warning(`${successCount} uploaded · ${failed.length} failed`)
      setFailedFiles(failed)
    } else {
      toast.error(
        failed.length === 1
          ? `Failed to upload "${failed[0].name}"`
          : `All ${failed.length} uploads failed`,
      )
      setFailedFiles(failed)
    }
  }, [upload, effectiveUploadPath])

  const handleFiles = useCallback((rawFiles: FileList | null) => {
    const all = Array.from(rawFiles ?? [])
    if (!all.length) return

    setDupeWarning(null)
    setFailedFiles([])

    // Pre-validate each file immediately — toast per invalid file, filter them out
    const valid: File[] = []
    for (const file of all) {
      const err = validateFile(file)
      if (err) {
        toast.error(`"${file.name}": ${err}`)
      } else {
        valid.push(file)
      }
    }
    if (!valid.length) return

    // Duplicate detection against current directory listing
    const existingNames = new Set((items ?? []).map((i) => i.name))
    const dupes = valid.map((f) => f.name).filter((n) => existingNames.has(n))

    if (dupes.length > 0) {
      setDupeWarning({ files: valid, dupes })
      return
    }

    doUpload(valid)
  }, [items, doUpload])

  function handleNewFolder() {
    const name = newFolderInput.trim()
    if (!name) return
    setUploadDestFolder(name)
    setNewFolderInput('')
    setPickerOpen(false)
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Breadcrumb */}
      <div className="flex items-center gap-1 flex-wrap">
        <button
          type="button"
          onClick={() => setPathStack([])}
          className={cn(
            'flex items-center gap-1 font-mono text-xs transition',
            pathStack.length === 0 ? 'text-ink font-bold' : 'text-ink/50 hover:text-ink',
          )}
        >
          <Home size={12} />
          {bucket}
        </button>
        {pathStack.map((segment, i) => (
          <span key={i} className="flex items-center gap-1">
            <ChevronRight size={12} className="text-ink/25" />
            <button
              type="button"
              onClick={() => navigateTo(i + 1)}
              className={cn(
                'font-mono text-xs transition',
                i === pathStack.length - 1 ? 'text-ink font-bold' : 'text-ink/50 hover:text-ink',
              )}
            >
              {segment}
            </button>
          </span>
        ))}
      </div>

      {/* Drop zone + grid */}
      <div
        className={cn(
          'relative rounded-xl transition-colors',
          isDragging && 'ring-2 ring-forest ring-offset-2',
        )}
        onDragOver={(e) => { e.preventDefault(); setIsDragging(true) }}
        onDragLeave={() => setIsDragging(false)}
        onDrop={(e) => {
          e.preventDefault()
          setIsDragging(false)
          handleFiles(e.dataTransfer.files)
        }}
      >
        {isDragging && (
          <div className="absolute inset-0 z-10 bg-forest/10 rounded-xl flex flex-col items-center justify-center gap-1 pointer-events-none">
            <p className="font-sans font-semibold text-forest">Drop to upload</p>
            <p className="font-mono text-xs text-forest/70">→ {destLabel}</p>
          </div>
        )}

        {isLoading && (
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
            {Array.from({ length: 8 }).map((_, i) => <SkeletonCard key={i} />)}
          </div>
        )}

        {isError && (
          <div className="flex items-center gap-3 py-12 text-brand-red">
            <AlertCircle size={16} className="shrink-0" />
            <p className="font-mono text-sm">Failed to load files. Check your connection and try again.</p>
          </div>
        )}

        {!isLoading && !isError && items?.length === 0 && (
          <div className="flex flex-col items-center gap-3 py-20 text-center">
            <ImageIcon size={36} className="text-ink/20" />
            <p className="font-sans font-semibold text-ink/40">This folder is empty</p>
            <p className="font-mono text-xs text-ink/30">Drop files here or use the upload button.</p>
          </div>
        )}

        {!isLoading && !isError && !!items?.length && (
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
            {items.map((item) => (
              <StorageItemCard
                key={item.name}
                item={item}
                bucket={bucket}
                folderPath={currentPath}
                selectable={selectable}
                onNavigate={navigateInto}
                onSelect={onSelect}
              />
            ))}
          </div>
        )}
      </div>

      {/* Upload bar */}
      <div className="flex flex-col gap-2 pt-2 border-t-2 border-ink/8">

        {/* Progress bar */}
        {uploadPercent !== null && (
          <div className="flex items-center gap-3">
            <div className="flex-1 h-1.5 bg-ink/8 rounded-full overflow-hidden">
              <div
                ref={progressFillRef}
                className="progress-bar-fill h-full bg-forest rounded-full transition-all duration-200"
              />
            </div>
            <span className="font-mono text-xs text-ink/50 tabular-nums shrink-0 w-8 text-right">
              {uploadPercent}%
            </span>
          </div>
        )}

        {/* Duplicate warning */}
        {dupeWarning && (
          <div className="flex items-start gap-2.5 border-2 border-amber-300 bg-amber-50 rounded-xl px-3 py-2.5">
            <AlertTriangle size={13} className="text-amber-500 shrink-0 mt-0.5" />
            <div className="flex-1 min-w-0">
              <p className="font-mono text-xs text-amber-800">
                <span className="font-bold">
                  {dupeWarning.dupes.length === 1
                    ? `"${dupeWarning.dupes[0]}"`
                    : `${dupeWarning.dupes.length} files`}
                </span>
                {' '}already exist and will be overwritten.
              </p>
              {dupeWarning.dupes.length > 1 && (
                <p className="font-mono text-[0.65rem] text-amber-600 mt-0.5 truncate">
                  {dupeWarning.dupes.join(', ')}
                </p>
              )}
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <button
                type="button"
                onClick={() => {
                  const skipped = dupeWarning.files.filter(
                    (f) => !dupeWarning.dupes.includes(f.name),
                  )
                  setDupeWarning(null)
                  if (skipped.length) doUpload(skipped)
                  else toast.info('No new files to upload.')
                }}
                className="font-mono text-[0.65rem] uppercase tracking-widest text-amber-700 hover:underline"
              >
                Skip
              </button>
              <button
                type="button"
                onClick={() => { const f = dupeWarning.files; setDupeWarning(null); doUpload(f) }}
                className="font-mono text-[0.65rem] uppercase tracking-widest text-ink font-bold border-2 border-ink/20 rounded-md px-2 py-0.5 hover:bg-ink hover:text-cream hover:border-ink transition"
              >
                Overwrite
              </button>
            </div>
          </div>
        )}

        {/* Per-file error list */}
        {failedFiles.length > 0 && (
          <div className="border-2 border-brand-red/20 bg-brand-red/5 rounded-xl px-3 py-2.5 flex flex-col gap-1.5">
            <div className="flex items-center justify-between">
              <p className="font-mono text-[0.65rem] uppercase tracking-widest text-brand-red font-bold">
                {failedFiles.length} upload{failedFiles.length > 1 ? 's' : ''} failed
              </p>
              <button
                type="button"
                onClick={() => setFailedFiles([])}
                className="text-ink/30 hover:text-ink transition"
                aria-label="Dismiss errors"
              >
                <X size={12} />
              </button>
            </div>
            {failedFiles.map((f) => (
              <div key={f.name} className="flex items-baseline gap-2">
                <span className="font-mono text-xs text-ink truncate max-w-40">{f.name}</span>
                <span className="font-mono text-[0.65rem] text-brand-red shrink-0">{f.reason}</span>
              </div>
            ))}
          </div>
        )}

        {/* Controls row */}
        <div className="flex items-center gap-3 flex-wrap">
          <label className={cn(
            'border-2 border-ink rounded-lg px-4 py-2 font-mono text-xs font-bold uppercase tracking-widest cursor-pointer transition shrink-0',
            isUploading
              ? 'opacity-50 pointer-events-none text-ink/50'
              : 'text-ink hover:bg-ink hover:text-cream',
          )}>
            {isUploading
              ? `Uploading${uploadPercent !== null ? ` ${uploadPercent}%` : '…'}`
              : 'Upload files'}
            <input
              type="file"
              multiple
              accept="image/jpeg,image/png,image/webp,image/gif,image/svg+xml,video/mp4,video/webm,video/ogg,video/quicktime"
              className="hidden"
              onChange={(e) => handleFiles(e.target.files)}
              disabled={isUploading}
            />
          </label>

          {/* Folder picker */}
          <div className="relative" ref={pickerRef}>
            <button
              type="button"
              onClick={() => setPickerOpen((v) => !v)}
              className={cn(
                'flex items-center gap-1.5 font-mono text-xs border-2 rounded-lg px-3 py-2 transition',
                pickerOpen
                  ? 'border-ink/40 text-ink bg-ink/4'
                  : 'border-ink/15 text-ink/50 hover:border-ink/30 hover:text-ink',
              )}
            >
              <span className="text-ink/30">↳</span>
              <span className="max-w-30 truncate">{destLabel}</span>
              <ChevronDown size={11} className={cn('transition-transform', pickerOpen && 'rotate-180')} />
            </button>

            {pickerOpen && (
              <div className="absolute bottom-full mb-2 left-0 min-w-52 bg-white border-2 border-ink/15 rounded-xl shadow-xl z-20 overflow-hidden">
                <button
                  type="button"
                  onClick={() => { setUploadDestFolder(''); setPickerOpen(false) }}
                  className="w-full text-left flex items-center gap-2 px-3 py-2.5 font-mono text-xs hover:bg-ink/4 transition"
                >
                  <span className="w-3 shrink-0">
                    {!uploadDestFolder && <Check size={11} className="text-forest" />}
                  </span>
                  <span className={cn(!uploadDestFolder && 'text-forest font-bold')}>
                    {currentPath ? pathStack.at(-1) : 'Root'}
                  </span>
                  <span className="text-ink/30 ml-auto">(current)</span>
                </button>

                {folders.length > 0 && (
                  <div className="border-t border-ink/6">
                    {folders.map((folder) => (
                      <button
                        key={folder.name}
                        type="button"
                        onClick={() => { setUploadDestFolder(folder.name); setPickerOpen(false) }}
                        className="w-full text-left flex items-center gap-2 px-3 py-2.5 font-mono text-xs hover:bg-ink/4 transition"
                      >
                        <span className="w-3 shrink-0">
                          {uploadDestFolder === folder.name && <Check size={11} className="text-forest" />}
                        </span>
                        <span className={cn(uploadDestFolder === folder.name && 'text-forest font-bold')}>
                          {folder.name}
                        </span>
                      </button>
                    ))}
                  </div>
                )}

                <div className="border-t border-ink/8 px-3 py-2.5 flex items-center gap-2">
                  <FolderPlus size={12} className="text-amber-400 shrink-0" />
                  <input
                    type="text"
                    value={newFolderInput}
                    onChange={(e) => setNewFolderInput(e.target.value.replace(/[^a-zA-Z0-9_-]/g, ''))}
                    placeholder="new-folder"
                    className="flex-1 font-mono text-xs bg-transparent outline-none placeholder:text-ink/25 min-w-0"
                    onKeyDown={(e) => { if (e.key === 'Enter') handleNewFolder() }}
                  />
                  <button
                    type="button"
                    disabled={!newFolderInput.trim()}
                    onClick={handleNewFolder}
                    className="font-mono text-[0.6rem] uppercase tracking-widest text-forest disabled:text-ink/20 hover:underline shrink-0"
                  >
                    Use
                  </button>
                </div>
              </div>
            )}
          </div>

          <p className="font-mono text-xs text-ink/40">Images up to 5 MB · MP4 · WebM up to 10 MB</p>
        </div>
      </div>
    </div>
  )
}
