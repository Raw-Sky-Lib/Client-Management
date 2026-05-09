import { cn } from '@/lib/utils'

interface AgencyBadgeProps {
  agency?: string
  className?: string
}

export function AgencyBadge({ agency = 'Format Studio', className }: AgencyBadgeProps) {
  return (
    <div className={cn('flex flex-col gap-3', className)}>
      <svg width="72" height="10" viewBox="0 0 100 10" preserveAspectRatio="none" aria-hidden>
        <path
          d="M0,5 L10,0 L20,10 L30,0 L40,10 L50,0 L60,10 L70,0 L80,10 L90,0 L100,5"
          fill="none"
          stroke="#E15A49"
          strokeWidth="3"
          strokeLinejoin="round"
        />
      </svg>

      <p className="font-mono text-[0.65rem] uppercase tracking-widest text-ink opacity-40">
        Built by
      </p>

      <div className="flex items-center gap-2.5">
        <div className="w-3 h-3 rounded-full bg-brand-red border-2 border-ink shrink-0" />
        <span className="font-sans font-extrabold text-xl text-ink tracking-tight leading-none">
          {agency}
        </span>
      </div>
    </div>
  )
}
