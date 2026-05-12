import { Link, useLocation } from 'react-router'
import {
  LayoutDashboard,
  FileText,
  BookOpen,
  Image,
  Inbox,
  Settings2,
  Sparkles,
  type LucideIcon,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { AgencyBadge } from '@/components/ui/agency-badge'

const NAV_ITEMS: { path: string; label: string; icon: LucideIcon }[] = [
  { path: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { path: '/pages',     label: 'Pages',     icon: FileText         },
  { path: '/blog',      label: 'Blog',      icon: BookOpen         },
  { path: '/media',     label: 'Media',     icon: Image            },
  { path: '/forms',     label: 'Forms',     icon: Inbox            },
  { path: '/settings',  label: 'Settings',  icon: Settings2        },
  { path: '/assistant', label: 'Assistant', icon: Sparkles         },
]

function isActive(pathname: string, itemPath: string): boolean {
  if (itemPath === '/dashboard') return pathname === '/dashboard'
  return pathname === itemPath || pathname.startsWith(itemPath + '/')
}

export function PortalSidebar() {
  const { pathname } = useLocation()

  return (
    <aside className="w-60 shrink-0 h-screen flex flex-col justify-between border-r-2 border-ink bg-white px-4 py-6">

      {/* Top: logo + nav */}
      <div className="flex flex-col gap-8">
        <div className="flex items-center gap-2.5 px-2 font-sans font-extrabold text-xl tracking-tight text-ink">
          <div className="w-3 h-3 rounded-full bg-brand-red border-2 border-ink shrink-0" />
          Client Portal
        </div>

        <nav>
          <ul className="flex flex-col gap-0.5">
            {NAV_ITEMS.map(({ path, label, icon: Icon }) => {
              const active = isActive(pathname, path)
              return (
                <li key={path}>
                  <Link
                    to={path}
                    className={cn(
                      'flex items-center gap-3 px-3 py-2.5 rounded-lg font-sans font-medium text-sm transition-colors',
                      active
                        ? 'bg-ink text-white'
                        : 'text-ink hover:bg-ink/10',
                    )}
                  >
                    <Icon size={16} className="shrink-0" />
                    {label}
                  </Link>
                </li>
              )
            })}
          </ul>
        </nav>
      </div>

      {/* Bottom: agency brand */}
      <AgencyBadge />
    </aside>
  )
}
