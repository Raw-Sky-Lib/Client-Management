import { Navigate, Outlet } from 'react-router'
import { useAuth } from '@/contexts/auth-context'

export function GuestRoute() {
  const { user, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div className="min-h-svh bg-cream flex items-center justify-center">
        <div className="w-5 h-5 rounded-full border-2 border-ink border-t-transparent animate-spin" />
      </div>
    )
  }

  if (user) return <Navigate to="/dashboard" replace />

  return <Outlet />
}
