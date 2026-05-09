import { type ReactNode } from 'react'
import { cn } from '@/lib/utils'

export interface OnboardingStep {
  label: string
  sublabel: string
  status: 'done' | 'active' | 'pending'
}

interface OnboardingLayoutProps {
  brandName?: string
  steps: OnboardingStep[]
  children: ReactNode
}

const SWATCHES = [
  { color: '#E15A49', label: 'brand-red' },
  { color: '#ECA7A2', label: 'brand-pink' },
  { color: '#F59E0B', label: 'brand-amber' },
  { color: '#1A3D2B', label: 'forest' },
]

function ZigZag() {
  return (
    <svg width="60" height="10" viewBox="0 0 100 10" preserveAspectRatio="none">
      <path
        d="M0,5 L10,0 L20,10 L30,0 L40,10 L50,0 L60,10 L70,0 L80,10 L90,0 L100,5"
        fill="none"
        stroke="#1C1C1A"
        strokeWidth="2"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function StepCircle({ status, index }: { status: OnboardingStep['status']; index: number }) {
  if (status === 'done') {
    return (
      <div className="w-6 h-6 rounded-full border-2 border-ink bg-ink text-white flex items-center justify-center font-mono text-[10px] font-bold mt-0.5 shrink-0">
        ✓
      </div>
    )
  }
  if (status === 'active') {
    return (
      <div className="w-6 h-6 rounded-full border-2 border-brand-red bg-brand-red text-white flex items-center justify-center font-mono text-[10px] font-bold mt-0.5 shrink-0">
        {index + 1}
      </div>
    )
  }
  return (
    <div className="w-6 h-6 rounded-full border-2 border-ink flex items-center justify-center font-mono text-[10px] font-bold mt-0.5 shrink-0">
      {index + 1}
    </div>
  )
}

export function OnboardingLayout({
  brandName = 'CLIENT_PORTAL',
  steps,
  children,
}: OnboardingLayoutProps) {
  return (
    <div className="min-h-svh bg-cream flex items-center justify-center p-8">
      <div className="w-full max-w-[1200px] relative" style={{ minHeight: '85vh' }}>

        {/* System header — spans full width above the grid */}
        <div className="absolute -top-10 left-0 right-0 flex justify-between font-mono text-xs uppercase tracking-widest text-ink opacity-60 select-none pointer-events-none">
          <span>CLIENT_PORTAL_OS</span>
          <span>BUILD_1.0</span>
          <span>STATUS_OK</span>
        </div>

        {/* Two-column grid */}
        <div
          className="grid gap-8 h-full"
          style={{ gridTemplateColumns: '320px 1fr', minHeight: '85vh' }}
        >
          {/* Sidebar */}
          <aside className="flex flex-col justify-between py-2">

            {/* Top: brand + steps */}
            <div>
              {/* Brand logo */}
              <div className="flex items-center gap-2 font-sans font-extrabold text-2xl tracking-tight mb-16 text-ink">
                <div className="w-3 h-3 rounded-full bg-brand-red border-2 border-ink shrink-0" />
                {brandName}
              </div>

              {/* Step list */}
              <ul className="flex flex-col gap-6">
                {steps.map((step, i) => (
                  <li
                    key={step.label}
                    className={cn(
                      'grid gap-4 items-start',
                      step.status === 'pending' && 'opacity-35',
                    )}
                    style={{ gridTemplateColumns: '24px 1fr' }}
                  >
                    <StepCircle status={step.status} index={i} />
                    <div>
                      <h3 className="font-sans font-bold text-[1.05rem] text-ink leading-tight mb-0.5">
                        {step.label}
                      </h3>
                      <p className="font-mono text-[0.75rem] text-ink opacity-70 uppercase tracking-wide">
                        {step.sublabel}
                      </p>
                    </div>
                  </li>
                ))}
              </ul>
            </div>

            {/* Sidebar footer */}
            <div className="flex flex-col gap-3">
              <ZigZag />
              <div className="flex gap-2">
                {SWATCHES.map((s) => (
                  <div
                    key={s.label}
                    className="w-6 h-6 rounded-[6px] border-[1.5px] border-ink"
                    style={{ backgroundColor: s.color }}
                  />
                ))}
              </div>
              <span className="font-mono text-xs uppercase tracking-widest text-ink opacity-50">
                CLIENT-PORTAL.APP
              </span>
            </div>
          </aside>

          {/* Main content slot */}
          <main className="flex flex-col">{children}</main>
        </div>
      </div>
    </div>
  )
}
