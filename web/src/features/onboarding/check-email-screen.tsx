import { HardShadowCard } from '@/components/ui/hard-shadow-card'
import { StatusPill } from '@/components/ui/status-pill'

interface CheckEmailScreenProps {
  email:    string
  variant?: 'login' | 'onboarding'
}

export function CheckEmailScreen({ email, variant = 'onboarding' }: CheckEmailScreenProps) {
  const isLogin = variant === 'login'

  return (
    <HardShadowCard className="flex flex-col flex-1 overflow-hidden">

      {/* Hero */}
      <div className="relative flex flex-col justify-between p-10 overflow-hidden [flex:0_0_58%] bg-[linear-gradient(135deg,#1A3D2B_0%,#0E2419_100%)]">

        {/* Grid overlay */}
        <div className="absolute inset-0 pointer-events-none z-[1] [background-image:linear-gradient(rgba(255,255,255,0.05)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.05)_1px,transparent_1px)] [background-size:30px_30px]" />

        {/* Amber circle */}
        <div className="absolute z-[2] w-80 h-80 -bottom-[20%] -right-[4%] flex items-center justify-center border-2 border-ink rounded-full bg-amber-400">
          <span className="font-sans font-extrabold text-ink select-none text-[8rem] leading-none -rotate-[10deg]">
            ✓
          </span>
        </div>

        {/* Headline */}
        <h1 className="relative z-[3] font-sans font-extrabold text-white uppercase leading-[1.05] tracking-tight text-[3.2rem] max-w-[70%]">
          Check your inbox.
        </h1>
      </div>

      {/* Body */}
      <div className="flex flex-col items-center justify-center flex-1 px-10 py-8 text-center gap-4">
        <StatusPill variant="active">Email sent</StatusPill>

        <div className="max-w-md">
          <h2 className="font-sans font-extrabold text-xl text-ink mb-2">
            {isLogin ? 'Sign-in link sent.' : 'Confirmation link sent.'}
          </h2>
          <p className="font-mono text-sm text-ink opacity-60 leading-relaxed">
            We sent a link to{' '}
            <span className="font-bold text-ink opacity-100">{email}</span>
            {'. '}
            {isLogin
              ? 'Click it to sign in to your dashboard.'
              : 'Click it to finish setting up your account.'}
          </p>
        </div>

        <p className="font-mono text-xs text-ink opacity-40 mt-2">
          Didn't get it? Check spam, or contact your website team.
        </p>
      </div>
    </HardShadowCard>
  )
}
