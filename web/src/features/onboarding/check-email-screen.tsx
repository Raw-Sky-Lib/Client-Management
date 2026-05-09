import { HardShadowCard } from '@/components/ui/hard-shadow-card'
import { StatusPill } from '@/components/ui/status-pill'

interface CheckEmailScreenProps {
  email: string
}

export function CheckEmailScreen({ email }: CheckEmailScreenProps) {
  return (
    <HardShadowCard className="flex flex-col flex-1 overflow-hidden">

      {/* Hero — forest gradient + amber circle, mirrors Variants success screen */}
      <div
        className="relative flex flex-col justify-between p-10 overflow-hidden"
        style={{
          background: 'linear-gradient(135deg, #1A3D2B 0%, #0E2419 100%)',
          flex: '0 0 58%',
        }}
      >
        {/* Subtle grid overlay */}
        <div
          className="absolute inset-0 pointer-events-none"
          style={{
            backgroundImage:
              'linear-gradient(rgba(255,255,255,0.05) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.05) 1px, transparent 1px)',
            backgroundSize: '30px 30px',
            zIndex: 1,
          }}
        />

        {/* Amber circle — partially clipped at bottom-right */}
        <div
          className="absolute flex items-center justify-center border-2 border-ink rounded-full"
          style={{
            width: 320,
            height: 320,
            bottom: '-20%',
            right: '-4%',
            backgroundColor: '#F59E0B',
            zIndex: 2,
          }}
        >
          <span
            className="font-sans font-extrabold text-ink select-none"
            style={{ fontSize: '8rem', lineHeight: 1, transform: 'rotate(-10deg)' }}
          >
            ✓
          </span>
        </div>

        {/* Headline */}
        <h1
          className="font-sans font-extrabold text-white uppercase leading-[1.05] tracking-tight"
          style={{ fontSize: '3.2rem', maxWidth: '70%', zIndex: 3 }}
        >
          Check your inbox.
        </h1>
      </div>

      {/* Body */}
      <div className="flex flex-col items-center justify-center flex-1 px-10 py-8 text-center gap-4">
        <StatusPill variant="active">Email sent</StatusPill>

        <div className="max-w-md">
          <h2 className="font-sans font-extrabold text-xl text-ink mb-2">
            Confirmation link sent.
          </h2>
          <p className="font-mono text-sm text-ink opacity-60 leading-relaxed">
            We sent a link to{' '}
            <span className="font-bold text-ink opacity-100">{email}</span>
            {'. '}
            Click it to finish setting up your account.
          </p>
        </div>

        <p className="font-mono text-xs text-ink opacity-40 mt-2">
          Didn't get it? Check spam, or contact your website team.
        </p>
      </div>
    </HardShadowCard>
  )
}
