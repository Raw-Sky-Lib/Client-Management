import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { isAxiosError } from 'axios'
import api from '@/lib/axios'
import { HardShadowCard } from '@/components/ui/hard-shadow-card'
import { cn } from '@/lib/utils'

const schema = z.object({
  access_code: z.string().min(1, 'Access code is required'),
  email:       z.string().email('Enter a valid email address'),
})

type FormValues = z.infer<typeof schema>

const ERROR_MESSAGES: Record<string, string> = {
  expired: 'Your access code has expired. Ask your website team for a new one.',
  used:    'This access code has already been used. Contact your website team.',
  invalid: 'Invalid access code. Check for typos and try again.',
}

function resolveError(raw: string): string {
  return ERROR_MESSAGES[raw] ?? (raw || 'Something went wrong. Please try again.')
}

interface ConnectFormProps {
  onSuccess: (email: string) => void
}

export function ConnectForm({ onSuccess }: ConnectFormProps) {
  const [formError, setFormError] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema) })

  async function onSubmit(values: FormValues) {
    setFormError(null)
    try {
      await api.post('/api/onboarding/connect', {
        token: values.access_code,
        email: values.email,
      })
      onSuccess(values.email)
    } catch (err) {
      if (isAxiosError(err)) {
        const raw = (err.response?.data?.error ?? err.response?.data?.message ?? '') as string
        setFormError(resolveError(raw))
      } else {
        setFormError('Something went wrong. Please try again.')
      }
    }
  }

  return (
    <HardShadowCard className="flex flex-col flex-1">

      {/* Card header */}
      <div className="px-10 pt-8 pb-6 border-b-2 border-ink">
        <h2 className="font-sans font-extrabold text-[2rem] leading-tight tracking-tight text-ink">
          Connect your workspace.
        </h2>
        <p className="font-mono text-sm text-ink opacity-60 mt-1">
          Enter the access code your website team sent you.
        </p>
      </div>

      {/* Form body */}
      <form
        onSubmit={handleSubmit(onSubmit)}
        className="flex flex-col flex-1 px-10 py-8 gap-6"
      >
        <div className="flex flex-col gap-5">

          {/* Access code */}
          <div className="flex flex-col gap-1.5">
            <label className="font-mono text-xs uppercase tracking-widest text-ink">
              Access Code
            </label>
            <input
              {...register('access_code')}
              disabled={isSubmitting}
              placeholder="xxxx-xxxx-xxxx"
              autoComplete="off"
              className={cn(
                'w-full border-2 border-ink rounded-lg px-4 py-3 font-sans text-base text-ink bg-white',
                'placeholder:text-ink/30 outline-none',
                'focus:ring-2 focus:ring-forest/20 focus:border-forest',
                'disabled:opacity-50 transition',
                errors.access_code && 'border-brand-red focus:ring-brand-red/20 focus:border-brand-red',
              )}
            />
            {errors.access_code && (
              <p className="font-mono text-xs text-brand-red">{errors.access_code.message}</p>
            )}
          </div>

          {/* Email */}
          <div className="flex flex-col gap-1.5">
            <label className="font-mono text-xs uppercase tracking-widest text-ink">
              Email Address
            </label>
            <input
              {...register('email')}
              type="email"
              disabled={isSubmitting}
              placeholder="you@yourcompany.com"
              autoComplete="email"
              className={cn(
                'w-full border-2 border-ink rounded-lg px-4 py-3 font-sans text-base text-ink bg-white',
                'placeholder:text-ink/30 outline-none',
                'focus:ring-2 focus:ring-forest/20 focus:border-forest',
                'disabled:opacity-50 transition',
                errors.email && 'border-brand-red focus:ring-brand-red/20 focus:border-brand-red',
              )}
            />
            {errors.email && (
              <p className="font-mono text-xs text-brand-red">{errors.email.message}</p>
            )}
          </div>
        </div>

        {/* Error banner */}
        {formError && (
          <div className="border-2 border-brand-red rounded-lg px-4 py-3" style={{ backgroundColor: 'rgba(225,90,73,0.08)' }}>
            <p className="font-mono text-xs text-brand-red">{formError}</p>
          </div>
        )}

        {/* Push action bar to bottom */}
        <div className="flex-1" />

        {/* Action bar */}
        <div className="border-t-2 border-dashed border-ink pt-6">
          <button
            type="submit"
            disabled={isSubmitting}
            className="btn-ink-shadow w-full flex items-center justify-center gap-3 bg-forest text-white font-sans font-extrabold text-lg uppercase tracking-wider py-4 px-8 rounded-lg border-2 border-ink disabled:opacity-60"
          >
            {isSubmitting ? (
              <>
                <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin shrink-0" />
                Sending…
              </>
            ) : (
              <>
                Connect to Portal
                <span className="text-xl leading-none">⇲</span>
              </>
            )}
          </button>
        </div>
      </form>
    </HardShadowCard>
  )
}
