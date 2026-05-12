import { useSearchParams, useNavigate } from 'react-router'
import { OnboardingLayout, type OnboardingStep } from '@/components/layout/onboarding-layout'
import { HardShadowCard } from '@/components/ui/hard-shadow-card'

const STEPS: OnboardingStep[] = [
  { label: 'Enter Code',     sublabel: 'Code verified',  status: 'done'    },
  { label: 'Verify Email',   sublabel: 'Link issue',     status: 'active'  },
  { label: 'Access Granted', sublabel: 'Pending',        status: 'pending' },
]

type Reason = 'used' | 'expired' | 'invalid' | 'error'

const MESSAGES: Record<Reason, { heading: string; body: string; cta: string; ctaPath: string }> = {
  used: {
    heading: 'Account already verified.',
    body:    'Your email has already been confirmed and your account is active. Sign in with your password to access your dashboard.',
    cta:     'Sign in →',
    ctaPath: '/login',
  },
  expired: {
    heading: 'Link expired.',
    body:    'This link is no longer valid. Confirmation links expire after 72 hours.',
    cta:     'Request a new link',
    ctaPath: '/connect',
  },
  invalid: {
    heading: 'Invalid link.',
    body:    'This link doesn\'t look right. It may have been copied incorrectly or is missing characters.',
    cta:     'Go back',
    ctaPath: '/login',
  },
  error: {
    heading: 'Something went wrong.',
    body:    'We couldn\'t complete your sign-in. Please try again or contact your website team.',
    cta:     'Try again',
    ctaPath: '/login',
  },
}

export function LinkErrorPage() {
  const [params] = useSearchParams()
  const navigate  = useNavigate()
  const reason    = (params.get('reason') ?? 'error') as Reason
  const msg       = MESSAGES[reason] ?? MESSAGES.error

  return (
    <OnboardingLayout steps={STEPS}>
      <HardShadowCard className="flex flex-col flex-1">

        {/* Header */}
        <div className="px-10 pt-8 pb-6 border-b-2 border-ink">
          <div className="w-8 h-8 rounded-full bg-brand-red border-2 border-ink flex items-center justify-center mb-4">
            <span className="text-white font-mono font-bold text-sm leading-none">✕</span>
          </div>
          <h2 className="font-sans font-extrabold text-[2rem] leading-tight tracking-tight text-ink">
            {msg.heading}
          </h2>
          <p className="font-mono text-sm text-ink opacity-60 mt-2">
            {msg.body}
          </p>
        </div>

        <div className="flex flex-col flex-1 px-10 py-8 gap-6">

          {/* Hint box */}
          <div className="border-2 border-ink/15 rounded-lg px-5 py-4 bg-cream">
            <p className="font-mono text-xs uppercase tracking-widest text-ink opacity-50 mb-2">
              What to do
            </p>
            {reason === 'used' && (
              <ul className="flex flex-col gap-1.5">
                <li className="font-mono text-sm text-ink opacity-70">→ Sign in with your password on the login page</li>
                <li className="font-mono text-sm text-ink opacity-70">→ Forgot your password? Use "Send a sign-in link" on the login page</li>
              </ul>
            )}
            {reason === 'expired' && (
              <ul className="flex flex-col gap-1.5">
                <li className="font-mono text-sm text-ink opacity-70">→ Ask your website team to resend the invite</li>
                <li className="font-mono text-sm text-ink opacity-70">→ New links are valid for 72 hours</li>
              </ul>
            )}
            {(reason === 'invalid' || reason === 'error') && (
              <ul className="flex flex-col gap-1.5">
                <li className="font-mono text-sm text-ink opacity-70">→ Check the link in your email is complete</li>
                <li className="font-mono text-sm text-ink opacity-70">→ Contact your website team if the issue persists</li>
              </ul>
            )}
          </div>

          <div className="flex-1" />

          {/* Action */}
          <div className="border-t-2 border-dashed border-ink pt-6 flex flex-col gap-3">
            <button
              type="button"
              onClick={() => navigate(msg.ctaPath)}
              className="btn-ink-shadow w-full flex items-center justify-center gap-3 bg-forest text-white font-sans font-extrabold text-lg uppercase tracking-wider py-4 px-8 rounded-lg border-2 border-ink"
            >
              {msg.cta}
              <span className="text-xl leading-none">⇲</span>
            </button>
          </div>
        </div>

      </HardShadowCard>
    </OnboardingLayout>
  )
}
