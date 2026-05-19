import { Navigate, Outlet } from 'react-router'
import { useAuth } from '@/contexts/auth-context'

// Auth check only — no SupabaseProvider. Used for pages that are protected
// but don't need a project loaded (e.g. /welcome, where a new client has no
// project yet and SupabaseProvider would block with "Failed to load project").
export function AuthOnlyRoute() {
  const { user, isLoading } = useAuth()

  if (isLoading) return null

  if (!user) return <Navigate to="/login" replace />

  return <Outlet />
}
