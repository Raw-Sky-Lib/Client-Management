import { Outlet, useLocation } from 'react-router'
import { PortalSidebar } from './portal-sidebar'
import { PortalHeader } from './portal-header'

const ROUTE_TITLES: Record<string, string> = {
  '/dashboard': 'Dashboard',
  '/pages':     'Pages',
  '/blog':      'Blog',
  '/blog/new':  'New Post',
  '/media':     'Media Library',
  '/forms':     'Forms',
  '/settings':  'Settings',
  '/assistant': 'Assistant',
}

function getTitle(pathname: string): string {
  if (ROUTE_TITLES[pathname]) return ROUTE_TITLES[pathname]
  if (pathname.startsWith('/pages/')) return 'Page Editor'
  if (pathname.startsWith('/blog/') && pathname.endsWith('/edit')) return 'Edit Post'
  return ''
}

export function PortalLayout() {
  const { pathname } = useLocation()

  return (
    <div className="flex h-screen overflow-hidden bg-cream">
      <PortalSidebar />
      <div className="flex flex-col flex-1 min-w-0">
        <PortalHeader title={getTitle(pathname)} />
        <main className="flex-1 overflow-y-auto p-8">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
