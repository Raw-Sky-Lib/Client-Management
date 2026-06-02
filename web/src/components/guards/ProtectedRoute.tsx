import { Navigate, Outlet } from 'react-router'
import { useAuth } from '@/contexts/auth-context'
import { SupabaseProvider } from '@/contexts/supabase-context'

export function ProtectedRoute() {
  const { user, isLoading } = useAuth()

  if (isLoading) return <AuthLoadingSpinner />

  if (!user) return <Navigate to="/login" replace />

  return (
    <SupabaseProvider>
      <Outlet />
    </SupabaseProvider>
  )
}

function AuthLoadingSpinner() {
  return (
    <div className="min-h-svh bg-cream flex items-center justify-center">
      <div className="w-5 h-5 rounded-full border-2 border-ink border-t-transparent animate-spin" />
    </div>
  )
}
