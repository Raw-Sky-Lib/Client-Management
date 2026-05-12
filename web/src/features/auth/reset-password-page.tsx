import { useState } from 'react'
import { isAxiosError } from 'axios'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useNavigate, useSearchParams } from 'react-router'
import { AgencyBadge } from '@/components/ui/agency-badge'
import { HardShadowCard } from '@/components/ui/hard-shadow-card'
import { cn } from '@/lib/utils'
import api from '@/lib/axios'
import { useAuth } from '@/contexts/auth-context'

const schema = z.object({
  password:        z.string().min(8, 'Password must be at least 8 characters'),
  confirmPassword: z.string().min(1, 'Please confirm your password'),
}).refine((d) => d.password === d.confirmPassword, {
  message: 'Passwords do not match',
  path: ['confirmPassword'],
})
type FormValues = z.infer<typeof schema>

export function ResetPasswordPage() {
  const [searchParams]    = useSearchParams()
  const navigate          = useNavigate()
  const { refresh }       = useAuth()
  const [formError, setFormError] = useState<string | null>(null)

  const token = searchParams.get('token')

  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<FormValues>({
    resolver: zodResolver(schema),
  })

  if (!token) {
    navigate('/link-error?reason=invalid', { replace: true })
    return null
  }

  async function onSubmit(values: FormValues) {
    setFormError(null)
    try {
      await api.post('/api/auth/reset-password/confirm', { token, password: values.password })
      await refresh()
      navigate('/dashboard', { replace: true })
    } catch (err) {
      if (isAxiosError(err)) {
        const status = err.response?.status
        if (status === 401) {
          setFormError('This reset link is invalid or has expired. Request a new one from the sign-in page.')
        } else {
          setFormError((err.response?.data?.error as string) || 'Something went wrong. Please try again.')
        }
      } else {
        setFormError('Something went wrong. Please try again.')
      }
    }
  }

  return (
    <div className="min-h-svh bg-cream flex flex-col items-center justify-center p-8 gap-8">

      <div className="flex items-center gap-2.5 font-sans font-extrabold text-2xl tracking-tight text-ink">
        <div className="w-3 h-3 rounded-full bg-brand-red border-2 border-ink shrink-0" />
        Client Portal
      </div>

      <div className="w-full max-w-md">
        <HardShadowCard>
          <div className="px-8 pt-7 pb-5 border-b-2 border-ink">
            <h2 className="font-sans font-extrabold text-[1.75rem] leading-tight tracking-tight text-ink">
              Choose a new password.
            </h2>
            <p className="font-mono text-sm text-ink opacity-60 mt-1">
              Must be at least 8 characters.
            </p>
          </div>

          <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col px-8 py-7 gap-5">
            <div className="flex flex-col gap-1.5">
              <label className="font-mono text-xs uppercase tracking-widest text-ink">
                New Password
              </label>
              <input
                {...register('password')}
                type="password"
                disabled={isSubmitting}
                placeholder="••••••••"
                autoComplete="new-password"
                className={cn(
                  'w-full border-2 border-ink rounded-lg px-4 py-3 font-sans text-base text-ink bg-white',
                  'placeholder:text-ink/30 outline-none',
                  'focus:ring-2 focus:ring-forest/20 focus:border-forest',
                  'disabled:opacity-50 transition',
                  errors.password && 'border-brand-red focus:ring-brand-red/20 focus:border-brand-red',
                )}
              />
              {errors.password && (
                <p className="font-mono text-xs text-brand-red">{errors.password.message}</p>
              )}
            </div>

            <div className="flex flex-col gap-1.5">
              <label className="font-mono text-xs uppercase tracking-widest text-ink">
                Confirm Password
              </label>
              <input
                {...register('confirmPassword')}
                type="password"
                disabled={isSubmitting}
                placeholder="••••••••"
                autoComplete="new-password"
                className={cn(
                  'w-full border-2 border-ink rounded-lg px-4 py-3 font-sans text-base text-ink bg-white',
                  'placeholder:text-ink/30 outline-none',
                  'focus:ring-2 focus:ring-forest/20 focus:border-forest',
                  'disabled:opacity-50 transition',
                  errors.confirmPassword && 'border-brand-red focus:ring-brand-red/20 focus:border-brand-red',
                )}
              />
              {errors.confirmPassword && (
                <p className="font-mono text-xs text-brand-red">{errors.confirmPassword.message}</p>
              )}
            </div>

            {formError && (
              <div className="border-2 border-brand-red rounded-lg px-4 py-3 bg-brand-red/8">
                <p className="font-mono text-xs text-brand-red">{formError}</p>
              </div>
            )}

            <div className="border-t-2 border-dashed border-ink pt-5 flex flex-col gap-3">
              <button
                type="submit"
                disabled={isSubmitting}
                className="btn-ink-shadow w-full flex items-center justify-center gap-3 bg-forest text-white font-sans font-extrabold text-lg uppercase tracking-wider py-4 px-8 rounded-lg border-2 border-ink disabled:opacity-60"
              >
                {isSubmitting ? (
                  <>
                    <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin shrink-0" />
                    Saving…
                  </>
                ) : (
                  <>
                    Set New Password
                    <span className="text-xl leading-none">⇲</span>
                  </>
                )}
              </button>

              <button
                type="button"
                onClick={() => navigate('/login')}
                disabled={isSubmitting}
                className="font-mono text-xs text-ink opacity-50 hover:opacity-80 transition text-center"
              >
                ← Back to sign in
              </button>
            </div>
          </form>
        </HardShadowCard>
      </div>

      <AgencyBadge />
    </div>
  )
}
