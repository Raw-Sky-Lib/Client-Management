import { useState } from 'react'
import { isAxiosError } from 'axios'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useNavigate } from 'react-router'
import { AgencyBadge } from '@/components/ui/agency-badge'
import { HardShadowCard } from '@/components/ui/hard-shadow-card'
import { cn } from '@/lib/utils'
import api from '@/lib/axios'
import { useAuth } from '@/contexts/auth-context'

// ─── Password login form ──────────────────────────────────────────────────────

const passwordSchema = z.object({
  email:    z.string().email('Enter a valid email address'),
  password: z.string().min(1, 'Password is required'),
})
type PasswordValues = z.infer<typeof passwordSchema>

function PasswordForm({ onReset }: { onReset: () => void }) {
  const [formError, setFormError] = useState<string | null>(null)
  const { refresh } = useAuth()
  const navigate    = useNavigate()

  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<PasswordValues>({
    resolver: zodResolver(passwordSchema),
  })

  async function onSubmit(values: PasswordValues) {
    setFormError(null)
    try {
      await api.post('/api/auth/login', { email: values.email, password: values.password })
      await refresh()
      navigate('/dashboard', { replace: true })
    } catch (err) {
      if (isAxiosError(err)) {
        const status = err.response?.status
        if (status === 401) {
          setFormError('Incorrect email or password.')
        } else {
          setFormError((err.response?.data?.error as string) || 'Something went wrong. Please try again.')
        }
      } else {
        setFormError('Something went wrong. Please try again.')
      }
    }
  }

  return (
    <HardShadowCard>
      <div className="px-8 pt-7 pb-5 border-b-2 border-ink">
        <h2 className="font-sans font-extrabold text-[1.75rem] leading-tight tracking-tight text-ink">
          Sign in to your dashboard.
        </h2>
        <p className="font-mono text-sm text-ink opacity-60 mt-1">
          Enter your email and password to continue.
        </p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col px-8 py-7 gap-5">
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

        <div className="flex flex-col gap-1.5">
          <label className="font-mono text-xs uppercase tracking-widest text-ink">
            Password
          </label>
          <input
            {...register('password')}
            type="password"
            disabled={isSubmitting}
            placeholder="••••••••"
            autoComplete="current-password"
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
                Signing in…
              </>
            ) : (
              <>
                Sign In
                <span className="text-xl leading-none">⇲</span>
              </>
            )}
          </button>

          <button
            type="button"
            onClick={onReset}
            disabled={isSubmitting}
            className="font-mono text-xs text-ink opacity-50 hover:opacity-80 transition text-center"
          >
            Forgot password? Reset it →
          </button>
        </div>

        <p className="font-mono text-xs text-ink opacity-40 text-center">
          First time?{' '}
          <a href="/connect" className="underline opacity-70 hover:opacity-100 transition">
            Use your access code
          </a>
        </p>
      </form>
    </HardShadowCard>
  )
}

// ─── Reset password request form ──────────────────────────────────────────────

const emailSchema = z.object({
  email: z.string().email('Enter a valid email address'),
})
type EmailValues = z.infer<typeof emailSchema>

function ResetRequestForm({ onBack }: { onBack: () => void }) {
  const [sent, setSent]         = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<EmailValues>({
    resolver: zodResolver(emailSchema),
  })

  async function onSubmit(values: EmailValues) {
    setFormError(null)
    try {
      await api.post('/api/auth/reset-password/request', { email: values.email })
      setSent(true)
    } catch (err) {
      if (isAxiosError(err)) {
        const msg = err.response?.data?.error as string
        if (err.response?.status === 429) {
          setFormError(msg || 'Please wait a moment before requesting another reset link.')
        } else {
          setFormError(msg || 'Something went wrong. Please try again.')
        }
      } else {
        setFormError('Something went wrong. Please try again.')
      }
    }
  }

  if (sent) {
    return (
      <HardShadowCard>
        <div className="px-8 pt-7 pb-5 border-b-2 border-ink">
          <h2 className="font-sans font-extrabold text-[1.75rem] leading-tight tracking-tight text-ink">
            Check your inbox.
          </h2>
          <p className="font-mono text-sm text-ink opacity-60 mt-1">
            If that email is registered, a reset link is on its way.
          </p>
        </div>
        <div className="px-8 py-7 flex flex-col gap-5">
          <p className="font-mono text-sm text-ink opacity-60 leading-relaxed">
            Click the link in the email to choose a new password. It expires in 1 hour.
          </p>
          <button
            type="button"
            onClick={onBack}
            className="font-mono text-xs text-ink opacity-50 hover:opacity-80 transition text-center"
          >
            ← Back to sign in
          </button>
        </div>
      </HardShadowCard>
    )
  }

  return (
    <HardShadowCard>
      <div className="px-8 pt-7 pb-5 border-b-2 border-ink">
        <h2 className="font-sans font-extrabold text-[1.75rem] leading-tight tracking-tight text-ink">
          Reset password.
        </h2>
        <p className="font-mono text-sm text-ink opacity-60 mt-1">
          Enter your email and we'll send you a reset link.
        </p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col px-8 py-7 gap-5">
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
                Sending…
              </>
            ) : (
              <>
                Send Reset Link
                <span className="text-xl leading-none">⇲</span>
              </>
            )}
          </button>

          <button
            type="button"
            onClick={onBack}
            disabled={isSubmitting}
            className="font-mono text-xs text-ink opacity-50 hover:opacity-80 transition text-center"
          >
            ← Back to sign in
          </button>
        </div>
      </form>
    </HardShadowCard>
  )
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export function LoginPage() {
  const [mode, setMode] = useState<'password' | 'reset'>('password')

  return (
    <div className="min-h-svh bg-cream flex flex-col items-center justify-center p-8 gap-8">

      <div className="flex items-center gap-2.5 font-sans font-extrabold text-2xl tracking-tight text-ink">
        <div className="w-3 h-3 rounded-full bg-brand-red border-2 border-ink shrink-0" />
        Client Portal
      </div>

      <div className="w-full max-w-md">
        {mode === 'password' ? (
          <PasswordForm onReset={() => setMode('reset')} />
        ) : (
          <ResetRequestForm onBack={() => setMode('password')} />
        )}
      </div>

      <AgencyBadge />
    </div>
  )
}
