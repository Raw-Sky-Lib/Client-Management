import { useState } from 'react'
import { Image, X } from 'lucide-react'
import { MediaPickerModal } from '@/features/media/components/media-picker-modal'

interface ImagePickerFieldProps {
  label: string
  value: string
  onChange: (url: string) => void
}

export function ImagePickerField({ label, value, onChange }: ImagePickerFieldProps) {
  const [pickerOpen, setPickerOpen] = useState(false)

  return (
    <>
      <div className="flex flex-col gap-1.5">
        <span className="font-mono text-[0.6rem] uppercase tracking-widest text-ink/50">{label}</span>

        <div className="flex items-center gap-3">
          {/* Thumbnail or placeholder */}
          <div className="w-16 h-16 shrink-0 rounded-md border-2 border-ink/15 overflow-hidden bg-ink/5 flex items-center justify-center">
            {value ? (
              <img
                src={value}
                alt=""
                className="w-full h-full object-cover"
                onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }}
              />
            ) : (
              <Image size={20} className="text-ink/20" />
            )}
          </div>

          {/* Actions */}
          <div className="flex flex-col gap-1.5 flex-1 min-w-0">
            <button
              type="button"
              onClick={() => setPickerOpen(true)}
              className="flex items-center gap-1.5 px-3 py-2 border-2 border-ink/20 rounded-lg font-mono text-xs text-ink/60 hover:border-ink/50 hover:text-ink transition text-left"
            >
              <Image size={12} />
              {value ? 'Change image' : 'Choose from media library'}
            </button>

            {value && (
              <button
                type="button"
                onClick={() => onChange('')}
                className="flex items-center gap-1 font-mono text-[0.6rem] text-brand-red/60 hover:text-brand-red transition"
              >
                <X size={10} />
                Clear
              </button>
            )}
          </div>
        </div>

        {/* URL preview */}
        {value && (
          <p className="font-mono text-[0.55rem] text-ink/30 truncate">{value}</p>
        )}
      </div>

      <MediaPickerModal
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        onSelect={(url) => { onChange(url); setPickerOpen(false) }}
      />
    </>
  )
}
