import { useState } from 'react'
import { isAxiosError } from 'axios'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { CheckCircle2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/axios'

const schema = z.object({
  password:        z.string().min(8, 'Password must be at least 8 characters'),
  confirmPassword: z.string().min(1, 'Please confirm your password'),
}).refine((d) => d.password === d.confirmPassword, {
  message: 'Passwords do not match',
  path: ['confirmPassword'],
})
type FormValues = z.infer<typeof schema>

export function SecuritySettings() {
  const [saved, setSaved]           = useState(false)
  const [formError, setFormError]   = useState<string | null>(null)

  const { register, handleSubmit, reset, formState: { errors, isSubmitting, isDirty } } = useForm<FormValues>({
    resolver: zodResolver(schema),
  })

  async function onSubmit(values: FormValues) {
    setFormError(null)
    setSaved(false)
    try {
      await api.post('/api/auth/set-password', { password: values.password })
      setSaved(true)
      reset()
    } catch (err) {
      if (isAxiosError(err)) {
        if (err.response?.status === 422) {
          setFormError('Your workspace isn\'t fully configured yet. Try again once your website team has finished the setup.')
        } else {
          setFormError(
            (err.response?.data?.error as string) || 'Could not update password. Please try again.',
          )
        }
      } else {
        setFormError('Something went wrong. Please try again.')
      }
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h2 className="font-sans font-bold text-base text-ink">Password</h2>
        <p className="font-mono text-xs text-ink/50 mt-0.5">
          Set or change the password you use to sign in to your dashboard.
        </p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
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
              'w-full border-2 border-ink rounded-lg px-4 py-3 font-sans text-sm text-ink bg-white',
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
              'w-full border-2 border-ink rounded-lg px-4 py-3 font-sans text-sm text-ink bg-white',
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

        {saved && (
          <div className="flex items-center gap-2 border-2 border-forest/30 rounded-lg px-4 py-3 bg-forest/5">
            <CheckCircle2 className="w-4 h-4 text-forest shrink-0" />
            <p className="font-mono text-xs text-forest">Password updated successfully.</p>
          </div>
        )}

        <div className="pt-2">
          <button
            type="submit"
            disabled={isSubmitting || !isDirty}
            className="btn-ink-shadow flex items-center justify-center gap-2 bg-ink text-white font-sans font-bold text-sm uppercase tracking-wider py-2.5 px-6 rounded-lg border-2 border-ink disabled:opacity-40 transition"
          >
            {isSubmitting ? (
              <>
                <span className="w-3.5 h-3.5 border-2 border-white/30 border-t-white rounded-full animate-spin shrink-0" />
                Saving…
              </>
            ) : (
              'Update Password'
            )}
          </button>
        </div>
      </form>
    </div>
  )
}
