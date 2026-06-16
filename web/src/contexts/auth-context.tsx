import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import api, { setCSRFToken } from '@/lib/axios'
import type { PortalUser } from '@/types'

interface AuthContextValue {
  user: PortalUser | null
  isLoading: boolean
  isAuthenticated: boolean
  logout: () => Promise<void>
  refresh: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<PortalUser | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    void bootstrap()
    async function bootstrap() {
      // CSRF must be set even when the user is unauthenticated — otherwise the
      // login / magic-link / reset-password forms can't attach the header and
      // every submission has to round-trip through the 403 retry path.
      try {
        const csrfRes = await api.get<{ csrf_token: string }>('/api/auth/csrf')
        setCSRFToken(csrfRes.data.csrf_token)
      } catch { /* recovered by axios 403 retry on first mutation */ }
      try {
        const profileRes = await api.get<PortalUser>('/api/auth/profile')
        setUser(profileRes.data)
      } catch {
        setUser(null)
      } finally {
        setIsLoading(false)
      }
    }
  }, [])

  async function refresh() {
    // Re-bootstrap CSRF first so the in-memory token reflects the latest cookie,
    // then fetch the profile. Same independence rule as the initial bootstrap.
    try {
      const csrfRes = await api.get<{ csrf_token: string }>('/api/auth/csrf')
      setCSRFToken(csrfRes.data.csrf_token)
    } catch { /* recoverable */ }
    try {
      const profileRes = await api.get<PortalUser>('/api/auth/profile')
      setUser(profileRes.data)
    } catch {
      setUser(null)
    }
  }

  async function logout() {
    await api.post('/api/auth/logout').catch(() => null)
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, isLoading, isAuthenticated: user !== null, logout, refresh }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used inside AuthProvider')
  }
  return ctx
}
