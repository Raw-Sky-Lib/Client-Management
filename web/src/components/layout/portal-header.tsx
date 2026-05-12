import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { useAuth } from '@/contexts/auth-context'

interface PortalHeaderProps {
  title: string
}

export function PortalHeader({ title }: PortalHeaderProps) {
  const { user, logout } = useAuth()
  const initial = user?.email?.[0]?.toUpperCase() ?? '?'

  return (
    <header className="h-14 shrink-0 flex items-center justify-between px-8 bg-cream border-b-2 border-ink">

      <h1 className="font-sans font-extrabold text-xl tracking-tight text-ink">
        {title}
      </h1>

      <div className="flex items-center gap-5">
        {/* View Site — enabled once site_url is wired in CLI-26+ */}
        <span
          className="font-mono text-xs text-ink opacity-30 select-none cursor-not-allowed"
          title="Available after first content save"
        >
          View Site ↗
        </span>

        {/* User menu */}
        <DropdownMenu.Root>
          <DropdownMenu.Trigger asChild>
            <button
              className="w-8 h-8 rounded-full bg-ink text-cream font-mono text-xs font-bold flex items-center justify-center border-2 border-ink focus:outline-none"
              aria-label="User menu"
            >
              {initial}
            </button>
          </DropdownMenu.Trigger>

          <DropdownMenu.Portal>
            <DropdownMenu.Content
              align="end"
              sideOffset={8}
              className="bg-white border-2 border-ink rounded-lg p-1 min-w-44 z-50"
              style={{ boxShadow: '6px 6px 0 #1C1C1A' }}
            >
              <div className="px-3 py-2 mb-1 border-b-2 border-ink/10">
                <p className="font-mono text-[0.7rem] text-ink opacity-50 truncate">
                  {user?.email}
                </p>
              </div>

              <DropdownMenu.Item
                onSelect={() => void logout()}
                className="flex items-center px-3 py-2 rounded-md font-sans text-sm text-brand-red cursor-pointer outline-none hover:bg-brand-red/10 transition-colors"
              >
                Sign out
              </DropdownMenu.Item>
            </DropdownMenu.Content>
          </DropdownMenu.Portal>
        </DropdownMenu.Root>
      </div>
    </header>
  )
}
