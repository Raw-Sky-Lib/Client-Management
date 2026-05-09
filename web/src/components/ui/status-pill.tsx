import { type ReactNode } from 'react'
import { cn } from '@/lib/utils'

type StatusVariant = 'active' | 'warning' | 'error'

const variantColor: Record<StatusVariant, string> = {
  active:  '#00C853',
  warning: '#F59E0B',
  error:   '#E15A49',
}

interface StatusPillProps {
  children: ReactNode
  variant?: StatusVariant
  className?: string
}

export function StatusPill({ children, variant = 'active', className }: StatusPillProps) {
  const color = variantColor[variant]
  return (
    <span
      className={cn(
        'inline-flex items-center gap-2 bg-ink rounded-full px-4 py-2 font-mono text-[0.7rem] font-bold uppercase tracking-widest',
        className,
      )}
      style={{ color }}
    >
      <span
        className="w-2 h-2 rounded-full shrink-0 animate-pulse"
        style={{ backgroundColor: color }}
      />
      {children}
    </span>
  )
}
