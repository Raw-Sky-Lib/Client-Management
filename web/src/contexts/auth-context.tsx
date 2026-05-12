import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import api from '@/lib/axios'
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
    // Bootstrap a fresh CSRF cookie first so every subsequent mutation request works.
    // If the access token is expired, the axios interceptor will silently refresh it
    // before retrying the profile call — no magic link needed until the 7-day refresh
    // token expires.
    api
      .get('/api/auth/csrf')
      .then(() => api.get<PortalUser>('/api/auth/profile'))
      .then((res) => setUser(res.data))
      .catch(() => setUser(null))
      .finally(() => setIsLoading(false))
  }, [])

  async function refresh() {
    try {
      const res = await api.get<PortalUser>('/api/auth/profile')
      setUser(res.data)
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
