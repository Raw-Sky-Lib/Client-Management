import { createClient, type SupabaseClient } from '@supabase/supabase-js'
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/axios'
import type { ProjectEntry } from '@/types'

interface ProjectsResponse {
  projects: ProjectEntry[]
}

interface SupabaseContextValue {
  supabase: SupabaseClient
  activeProject: ProjectEntry
  projects: ProjectEntry[]
  setActiveProjectId: (id: string) => void
}

const SupabaseContext = createContext<SupabaseContextValue | null>(null)

export function SupabaseProvider({ children }: { children: ReactNode }) {
  const [activeProjectId, setActiveProjectIdRaw] = useState<string | null>(null)
  const queryClient = useQueryClient()

  const setActiveProjectId = useCallback((id: string) => {
    setActiveProjectIdRaw(id)
    // Clear all tenant-scoped cached data so the new project's data loads fresh.
    queryClient.removeQueries({ predicate: (q) => q.queryKey[0] !== 'projects' })
  }, [queryClient])

  const { data, isLoading, isError } = useQuery({
    queryKey: ['projects'],
    queryFn: async () => {
      const res = await api.get<ProjectsResponse>('/api/projects')
      return res.data.projects
    },
    staleTime: 5 * 60 * 1000,
    retry: 2,
  })

  const projects = data ?? []

  const activeProject = useMemo(() => {
    if (projects.length === 0) return null
    return projects.find((p) => p.id === activeProjectId) ?? projects[0]
  }, [projects, activeProjectId])

  const supabase = useMemo(() => {
    if (!activeProject) return null
    return createClient(activeProject.supabase_url, activeProject.supabase_anon_key)
  }, [activeProject?.supabase_url, activeProject?.supabase_anon_key])

  if (isLoading) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-white">
        <div className="w-5 h-5 rounded-full border-2 border-ink border-t-transparent animate-spin" />
      </div>
    )
  }

  if (isError || !activeProject || !supabase) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-white">
        <p className="text-sm text-gray-500">Failed to load project. Please refresh the page.</p>
      </div>
    )
  }

  return (
    <SupabaseContext.Provider value={{ supabase, activeProject, projects, setActiveProjectId }}>
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

export function useProjectContext(): SupabaseContextValue {
  const ctx = useContext(SupabaseContext)
  if (!ctx) {
    throw new Error('useProjectContext must be used inside SupabaseProvider (inside ProtectedRoute)')
  }
  return ctx
}
