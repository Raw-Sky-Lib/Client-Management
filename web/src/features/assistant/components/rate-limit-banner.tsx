import { AlertTriangle, Clock, DollarSign } from 'lucide-react'
import type { RateLimitType } from '../hooks/use-assistant'

interface RateLimitBannerProps {
  type: RateLimitType
  message: string
}

const config: Record<RateLimitType, { icon: typeof Clock; label: string }> = {
  minute: { icon: Clock,        label: 'Slow down' },
  hour:   { icon: Clock,        label: 'Hourly limit' },
  budget: { icon: DollarSign,   label: 'Monthly limit' },
}

export function RateLimitBanner({ type, message }: RateLimitBannerProps) {
  const { icon: Icon, label } = config[type]

  return (
    <div className="flex items-start gap-3 border-2 border-amber-300 bg-amber-50 rounded-xl px-4 py-3">
      <AlertTriangle size={14} className="text-amber-500 shrink-0 mt-0.5" />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-0.5">
          <Icon size={11} className="text-amber-500" />
          <span className="font-mono text-[0.65rem] uppercase tracking-widest text-amber-600 font-bold">
            {label}
          </span>
        </div>
        <p className="font-mono text-xs text-amber-800">{message}</p>
      </div>
    </div>
  )
}
