import { Navigate, Outlet } from 'react-router'
import { useAuth } from '@/contexts/auth-context'
import { SupabaseProvider } from '@/contexts/supabase-context'

export function ProtectedRoute() {
  const { user, isLoading } = useAuth()

  if (isLoading) return null

  if (!user) return <Navigate to="/login" replace />

  return (
    <SupabaseProvider>
      <Outlet />
    </SupabaseProvider>
  )
}
