import { type ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { AgencyBadge } from '@/components/ui/agency-badge'


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
  brandName = 'Client Portal',
  steps,
  children,
}: OnboardingLayoutProps) {
  return (
    <div className="min-h-svh bg-cream flex items-center justify-center p-8">
      <div className="w-full max-w-300 grid grid-cols-[300px_1fr] gap-8 min-h-[85vh]">
        {/* Sidebar */}
        <aside className="flex flex-col justify-between py-2">

          {/* Top: brand + steps */}
          <div className="flex flex-col gap-12">
            {/* Brand */}
            <div className="flex items-center gap-2.5 font-sans font-extrabold text-2xl tracking-tight text-ink">
              <div className="w-3 h-3 rounded-full bg-brand-red border-2 border-ink shrink-0" />
              {brandName}
            </div>

            {/* Step list */}
            <ul className="flex flex-col gap-6">
              {steps.map((step, i) => (
                <li
                  key={step.label}
                  className={cn(
                    'grid grid-cols-[24px_1fr] gap-4 items-start',
                    step.status === 'pending' && 'opacity-35',
                  )}
                >
                  <StepCircle status={step.status} index={i} />
                  <div>
                    <h3 className="font-sans font-bold text-[1.05rem] text-ink leading-tight mb-0.5">
                      {step.label}
                    </h3>
                    <p className="font-mono text-[0.75rem] text-ink opacity-60">
                      {step.sublabel}
                    </p>
                  </div>
                </li>
              ))}
            </ul>
          </div>

          {/* Footer: agency brand */}
          <AgencyBadge />
        </aside>

        {/* Main content slot */}
        <main className="flex flex-col">{children}</main>
      </div>
    </div>
  )
}
