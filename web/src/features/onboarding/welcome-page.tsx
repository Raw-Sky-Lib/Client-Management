import { useState } from 'react'
import { isAxiosError } from 'axios'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useNavigate } from 'react-router'
import { FileText, PenLine, Image, Inbox, Settings2, Sparkles } from 'lucide-react'
import { OnboardingLayout, type OnboardingStep } from '@/components/layout/onboarding-layout'
import { HardShadowCard } from '@/components/ui/hard-shadow-card'
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

const STEPS_SET_PASSWORD: OnboardingStep[] = [
  { label: 'Invite Sent',     sublabel: 'Link delivered',    status: 'done'    },
  { label: 'Email Confirmed', sublabel: 'Identity verified', status: 'done'    },
  { label: 'Set Password',    sublabel: 'Choose a password', status: 'active'  },
  { label: 'Access Granted',  sublabel: 'Workspace ready',   status: 'pending' },
]

const STEPS_READY: OnboardingStep[] = [
  { label: 'Invite Sent',     sublabel: 'Link delivered',    status: 'done'   },
  { label: 'Email Confirmed', sublabel: 'Identity verified', status: 'done'   },
  { label: 'Set Password',    sublabel: 'Password saved',    status: 'done'   },
  { label: 'Access Granted',  sublabel: 'Workspace ready',   status: 'active' },
]

const FEATURES = [
  { icon: FileText,  label: 'Pages',             desc: 'Edit website sections and content'   },
  { icon: PenLine,   label: 'Blog',              desc: 'Publish and manage blog posts'        },
  { icon: Image,     label: 'Media',             desc: 'Upload and organise images and files' },
  { icon: Inbox,     label: 'Forms',             desc: 'View contact form submissions'        },
  { icon: Settings2, label: 'Settings',          desc: 'Site name, SEO and navigation'        },
  { icon: Sparkles,  label: 'Content Assistant', desc: 'AI-powered copy suggestions'          },
]

export function WelcomePage() {
  const navigate = useNavigate()
  const [phase, setPhase]           = useState<'set-password' | 'ready'>('set-password')
  const [noProject, setNoProject]   = useState(false)
  const [formError, setFormError]   = useState<string | null>(null)

  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<FormValues>({
    resolver: zodResolver(schema),
  })

  async function onSubmit(values: FormValues) {
    setFormError(null)
    try {
      await api.post('/api/auth/set-password', { password: values.password })
      setPhase('ready')
    } catch (err) {
      if (isAxiosError(err)) {
        if (err.response?.status === 422) {
          // No project configured yet — skip gracefully, client can set password later via reset flow
          setNoProject(true)
          setPhase('ready')
        } else {
          setFormError(
            (err.response?.data?.error as string) || 'Could not set password. Please try again.',
          )
        }
      } else {
        setFormError('Something went wrong. Please try again.')
      }
    }
  }

  if (phase === 'ready') {
    return (
      <OnboardingLayout steps={STEPS_READY}>
        <HardShadowCard className="flex flex-col flex-1">

          <div className="px-10 pt-8 pb-6 border-b-2 border-ink">
            <h2 className="font-sans font-extrabold text-[2rem] leading-tight tracking-tight text-ink">
              Your portal is ready.
            </h2>
            <p className="font-mono text-sm text-ink opacity-60 mt-1">
              Head to your dashboard to start managing your site.
            </p>
          </div>

          <div className="flex flex-col flex-1 px-10 py-8 gap-8 overflow-y-auto">

            {noProject && (
              <div className="border-2 border-ink/15 rounded-lg px-5 py-4 bg-cream">
                <p className="font-mono text-xs text-ink opacity-60 leading-relaxed">
                  → Your workspace is still being configured. You can set a password from the sign-in page once it's ready.
                </p>
              </div>
            )}

            <div className="flex flex-col gap-2">
              <p className="font-mono text-xs uppercase tracking-widest text-ink opacity-50">
                What you can manage
              </p>
              <div className="grid grid-cols-2 gap-3">
                {FEATURES.map(({ icon: Icon, label, desc }) => (
                  <div
                    key={label}
                    className="flex items-start gap-3 border-2 border-ink/15 rounded-lg px-4 py-3"
                  >
                    <Icon className="w-4 h-4 shrink-0 text-ink mt-0.5" />
                    <div>
                      <p className="font-sans font-bold text-sm text-ink leading-tight">{label}</p>
                      <p className="font-mono text-[0.7rem] text-ink opacity-50 mt-0.5">{desc}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div className="flex-1" />

            <div className="border-t-2 border-dashed border-ink pt-6">
              <button
                type="button"
                onClick={() => navigate('/dashboard')}
                className="btn-ink-shadow w-full flex items-center justify-center gap-3 bg-forest text-white font-sans font-extrabold text-lg uppercase tracking-wider py-4 px-8 rounded-lg border-2 border-ink"
              >
                Go to my dashboard
                <span className="text-xl leading-none">⇲</span>
              </button>
            </div>

          </div>
        </HardShadowCard>
      </OnboardingLayout>
    )
  }

  return (
    <OnboardingLayout steps={STEPS_SET_PASSWORD}>
      <HardShadowCard className="flex flex-col flex-1">

        <div className="px-10 pt-8 pb-6 border-b-2 border-ink">
          <h2 className="font-sans font-extrabold text-[2rem] leading-tight tracking-tight text-ink">
            Set your password.
          </h2>
          <p className="font-mono text-sm text-ink opacity-60 mt-1">
            You'll use this to sign in to your dashboard in the future.
          </p>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col flex-1 px-10 py-8 gap-6">

          <div className="flex flex-col gap-5">
            <div className="flex flex-col gap-1.5">
              <label className="font-mono text-xs uppercase tracking-widest text-ink">
                Password
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
          </div>

          {formError && (
            <div className="border-2 border-brand-red rounded-lg px-4 py-3 bg-brand-red/8">
              <p className="font-mono text-xs text-brand-red">{formError}</p>
            </div>
          )}

          <div className="flex-1" />

          <div className="border-t-2 border-dashed border-ink pt-6">
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
                  Set Password
                  <span className="text-xl leading-none">⇲</span>
                </>
              )}
            </button>
          </div>

        </form>
      </HardShadowCard>
    </OnboardingLayout>
  )
}
