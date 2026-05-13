import { useEffect } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import { useMarkRead } from '../hooks/use-submissions'
import { formatDate } from '@/lib/utils'
import type { FormSubmission } from '@/types'

interface SubmissionDetailProps {
  submission: FormSubmission | null
  onClose: () => void
}

export function SubmissionDetail({ submission, onClose }: SubmissionDetailProps) {
  const { mutate: markRead } = useMarkRead()

  // Mark as read when drawer opens on an unread submission
  useEffect(() => {
    if (submission && !submission.is_read) {
      markRead(submission.id)
    }
  }, [submission?.id])

  return (
    <Dialog.Root open={!!submission} onOpenChange={(open) => { if (!open) onClose() }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-ink/30 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />

        <Dialog.Content className="fixed right-0 top-0 bottom-0 z-50 w-full max-w-md bg-cream border-l-2 border-ink flex flex-col shadow-xl data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:slide-out-to-right data-[state=open]:slide-in-from-right duration-200">

          {/* Header */}
          <div className="flex items-center justify-between px-6 py-4 border-b-2 border-ink bg-ink shrink-0">
            <div>
              <Dialog.Title className="font-sans font-extrabold text-white">
                {submission?.form_name}
              </Dialog.Title>
              {submission && (
                <p className="font-mono text-[0.65rem] text-white/40 mt-0.5">
                  {formatDate(submission.submitted_at)}
                </p>
              )}
            </div>
            <Dialog.Close asChild>
              <button
                type="button"
                aria-label="Close"
                className="p-1.5 rounded-lg hover:bg-white/10 transition text-white/60 hover:text-white"
              >
                <X size={16} />
              </button>
            </Dialog.Close>
          </div>

          {/* Data */}
          <div className="flex-1 overflow-y-auto px-6 py-6">
            {submission && (
              <dl className="flex flex-col gap-4">
                {Object.entries(submission.data).map(([key, value]) => (
                  <div key={key} className="flex flex-col gap-1">
                    <dt className="font-mono text-[0.65rem] uppercase tracking-widest text-ink/40">
                      {key.replace(/_/g, ' ')}
                    </dt>
                    <dd className="font-sans text-sm text-ink bg-white border-2 border-ink/10 rounded-xl px-4 py-3 break-words whitespace-pre-wrap">
                      {String(value) || <span className="italic text-ink/30">empty</span>}
                    </dd>
                  </div>
                ))}
              </dl>
            )}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
