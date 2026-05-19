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
    // Poll every 10s when no project is configured yet so the UI updates
    // automatically once the agency pushes credentials.
    refetchInterval: (query) =>
      (query.state.data?.length ?? 0) === 0 ? 10_000 : false,
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

  if (isError) {
    return (
      <div className="flex min-h-svh items-center justify-center bg-white">
        <p className="text-sm text-gray-500">Failed to load project. Please refresh the page.</p>
      </div>
    )
  }

  if (!activeProject || !supabase) {
    return (
      <div className="flex min-h-svh flex-col items-center justify-center gap-4 bg-white px-6 text-center">
        <div className="w-10 h-10 rounded-full border-2 border-gray-200 flex items-center justify-center">
          <svg className="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
        </div>
        <div>
          <p className="font-semibold text-gray-800">No project set up yet</p>
          <p className="text-sm text-gray-500 mt-1">Your workspace is being prepared. Contact your team for access.</p>
        </div>
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
