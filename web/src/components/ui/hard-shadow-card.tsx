import { type ReactNode } from 'react'
import { cn } from '@/lib/utils'

const shadowMap = {
  lg: '8px 8px 0 #1C1C1A',
  md: '6px 6px 0 #1C1C1A',
  sm: '4px 4px 0 #1C1C1A',
}

interface HardShadowCardProps {
  children: ReactNode
  className?: string
  shadow?: keyof typeof shadowMap
}

export function HardShadowCard({ children, className, shadow = 'lg' }: HardShadowCardProps) {
  return (
    <div
      className={cn('bg-white border-2 border-ink rounded-card', className)}
      style={{ boxShadow: shadowMap[shadow] }}
    >
      {children}
    </div>
  )
}
