import { createClient, SupabaseClient } from '@supabase/supabase-js'
import { createContext, useContext, useMemo, type ReactNode } from 'react'

interface SupabaseContextValue {
  supabase: SupabaseClient
}

const SupabaseContext = createContext<SupabaseContextValue | null>(null)

interface SupabaseProviderProps {
  supabaseUrl: string
  supabaseAnonKey: string
  children: ReactNode
}

export function SupabaseProvider({ supabaseUrl, supabaseAnonKey, children }: SupabaseProviderProps) {
  const supabase = useMemo(
    () => createClient(supabaseUrl, supabaseAnonKey),
    [supabaseUrl, supabaseAnonKey],
  )

  return (
    <SupabaseContext.Provider value={{ supabase }}>
      {children}
    </SupabaseContext.Provider>
  )
}

export function useTenantSupabase(): SupabaseClient {
  const ctx = useContext(SupabaseContext)
  if (!ctx) {
    throw new Error('useTenantSupabase must be used inside SupabaseProvider (inside ProtectedRoute)')
  }
  return ctx.supabase
}
