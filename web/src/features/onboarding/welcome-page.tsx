import { useState } from 'react'
import { useNavigate } from 'react-router'
import { isAxiosError } from 'axios'
import { Database, FileText, PenLine, Image, Inbox, Settings2, Sparkles, CheckCircle2 } from 'lucide-react'
import { OnboardingLayout, type OnboardingStep } from '@/components/layout/onboarding-layout'
import { HardShadowCard } from '@/components/ui/hard-shadow-card'
import { useAuth } from '@/contexts/auth-context'
import { cn } from '@/lib/utils'
import api from '@/lib/axios'

const STEPS: OnboardingStep[] = [
  { label: 'Enter Code',     sublabel: 'Code verified',   status: 'done'   },
  { label: 'Verify Email',   sublabel: 'Email confirmed', status: 'done'   },
  { label: 'Access Granted', sublabel: 'Workspace ready', status: 'active' },
]

const FEATURES = [
  { icon: FileText,  label: 'Pages',             desc: 'Edit website sections and content'   },
  { icon: PenLine,   label: 'Blog',              desc: 'Publish and manage blog posts'        },
  { icon: Image,     label: 'Media',             desc: 'Upload and organise images and files' },
  { icon: Inbox,     label: 'Forms',             desc: 'View contact form submissions'        },
  { icon: Settings2, label: 'Settings',          desc: 'Site name, SEO and navigation'        },
  { icon: Sparkles,  label: 'Content Assistant', desc: 'AI-powered copy suggestions'          },
]

function supabaseRef(url: string): string {
  try {
    return new URL(url).hostname.split('.')[0]
  } catch {
    return url
  }
}

export function WelcomePage() {
  const { user }  = useAuth()
  const navigate  = useNavigate()
  const projectRef = user ? supabaseRef(user.supabase_url) : '…'

  const [password, setPassword]   = useState('')
  const [confirm, setConfirm]     = useState('')
  const [saving, setSaving]       = useState(false)
  const [saved, setSaved]         = useState(false)
  const [pwError, setPwError]     = useState<string | null>(null)

  async function handleSetPassword() {
    setPwError(null)
    if (password.length < 8) {
      setPwError('Password must be at least 8 characters.')
      return
    }
    if (password !== confirm) {
      setPwError('Passwords do not match.')
      return
    }
    setSaving(true)
    try {
      await api.post('/api/auth/set-password', { password })
      setSaved(true)
    } catch (err) {
      if (isAxiosError(err)) {
        setPwError(err.response?.data?.error as string || 'Could not set password. Please try again.')
      } else {
        setPwError('Something went wrong. Please try again.')
      }
    } finally {
      setSaving(false)
    }
  }

  return (
    <OnboardingLayout steps={STEPS}>
      <HardShadowCard className="flex flex-col flex-1">

        {/* Header */}
        <div className="px-10 pt-8 pb-6 border-b-2 border-ink">
          <h2 className="font-sans font-extrabold text-[2rem] leading-tight tracking-tight text-ink">
            Your portal is ready.
          </h2>
          <p className="font-mono text-sm text-ink opacity-60 mt-1">
            Set a password to sign in anytime, then head to your dashboard.
          </p>
        </div>

        <div className="flex flex-col flex-1 px-10 py-8 gap-8 overflow-y-auto">

          {/* Connected project */}
          <div className="flex flex-col gap-2">
            <p className="font-mono text-xs uppercase tracking-widest text-ink opacity-50">
              Connected Supabase Project
            </p>
            <div className="flex items-center gap-3 border-2 border-ink rounded-lg px-4 py-3 bg-cream">
              <Database className="w-4 h-4 shrink-0 text-ink" />
              <div className="min-w-0">
                <p className="font-mono text-sm font-bold text-ink">{projectRef}</p>
                <p className="font-mono text-xs text-ink opacity-50 truncate">
                  {user?.supabase_url}
                </p>
              </div>
              <div className="ml-auto flex items-center gap-1.5 shrink-0">
                <div className="w-2 h-2 rounded-full bg-forest" />
                <span className="font-mono text-[0.65rem] text-forest font-bold uppercase tracking-wider">Live</span>
              </div>
            </div>
          </div>

          {/* Features */}
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

          {/* Password setup */}
          <div className="flex flex-col gap-3">
            <p className="font-mono text-xs uppercase tracking-widest text-ink opacity-50">
              Set Your Password
            </p>
            <p className="font-mono text-sm text-ink opacity-60">
              Create a password so you can sign in at any time — no email link needed.
            </p>

            {saved ? (
              <div className="flex items-center gap-3 border-2 border-forest rounded-lg px-4 py-3 bg-forest/5">
                <CheckCircle2 className="w-4 h-4 text-forest shrink-0" />
                <p className="font-mono text-sm text-forest font-bold">Password set successfully.</p>
              </div>
            ) : (
              <div className="flex flex-col gap-3">
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  disabled={saving}
                  placeholder="Password (min. 8 characters)"
                  autoComplete="new-password"
                  className={cn(
                    'w-full border-2 border-ink rounded-lg px-4 py-3 font-sans text-base text-ink bg-white',
                    'placeholder:text-ink/30 outline-none',
                    'focus:ring-2 focus:ring-forest/20 focus:border-forest',
                    'disabled:opacity-50 transition',
                    pwError && 'border-brand-red focus:ring-brand-red/20 focus:border-brand-red',
                  )}
                />
                <input
                  type="password"
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  disabled={saving}
                  placeholder="Confirm password"
                  autoComplete="new-password"
                  className={cn(
                    'w-full border-2 border-ink rounded-lg px-4 py-3 font-sans text-base text-ink bg-white',
                    'placeholder:text-ink/30 outline-none',
                    'focus:ring-2 focus:ring-forest/20 focus:border-forest',
                    'disabled:opacity-50 transition',
                    pwError && 'border-brand-red focus:ring-brand-red/20 focus:border-brand-red',
                  )}
                />
                {pwError && (
                  <p className="font-mono text-xs text-brand-red">{pwError}</p>
                )}
                <button
                  type="button"
                  onClick={handleSetPassword}
                  disabled={saving || !password || !confirm}
                  className="btn-ink-shadow w-full flex items-center justify-center gap-3 bg-ink text-white font-sans font-bold text-base uppercase tracking-wider py-3 px-6 rounded-lg border-2 border-ink disabled:opacity-50"
                >
                  {saving ? (
                    <>
                      <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin shrink-0" />
                      Setting password…
                    </>
                  ) : (
                    'Set Password'
                  )}
                </button>
              </div>
            )}
          </div>

          <div className="flex-1" />

          {/* CTA */}
          <div className="border-t-2 border-dashed border-ink pt-6">
            <button
              type="button"
              onClick={() => navigate('/dashboard')}
              className="btn-ink-shadow w-full flex items-center justify-center gap-3 bg-forest text-white font-sans font-extrabold text-lg uppercase tracking-wider py-4 px-8 rounded-lg border-2 border-ink"
            >
              Go to my dashboard
              <span className="text-xl leading-none">⇲</span>
            </button>
            {!saved && (
              <p className="font-mono text-xs text-ink opacity-40 text-center mt-3">
                You can also set your password later from account settings.
              </p>
            )}
          </div>

        </div>
      </HardShadowCard>
    </OnboardingLayout>
  )
}
