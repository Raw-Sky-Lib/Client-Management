import { Navigate, Outlet } from 'react-router'
import { useAuth } from '@/contexts/auth-context'

export function GuestRoute() {
  const { user, isLoading } = useAuth()

  if (isLoading) return null

  if (user) return <Navigate to="/dashboard" replace />

  return <Outlet />
}
