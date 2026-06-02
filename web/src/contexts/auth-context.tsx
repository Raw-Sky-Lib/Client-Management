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
    // Bootstrap CSRF token (stored in memory) and session in parallel.
    // Profile is a GET — it doesn't need CSRF — so both can fire together.
    Promise.all([
      api.get<{ csrf_token: string }>('/api/auth/csrf'),
      api.get<PortalUser>('/api/auth/profile'),
    ])
      .then(([csrfRes, profileRes]) => {
        setCSRFToken(csrfRes.data.csrf_token)
        setUser(profileRes.data)
      })
      .catch(() => setUser(null))
      .finally(() => setIsLoading(false))
  }, [])

  async function refresh() {
    try {
      // Re-bootstrap CSRF and session together — ensures mutations work
      // immediately after the call (e.g. right after login/reset).
      const [csrfRes, profileRes] = await Promise.all([
        api.get<{ csrf_token: string }>('/api/auth/csrf'),
        api.get<PortalUser>('/api/auth/profile'),
      ])
      setCSRFToken(csrfRes.data.csrf_token)
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
