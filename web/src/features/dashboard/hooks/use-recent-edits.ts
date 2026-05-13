import { useQuery } from '@tanstack/react-query'
import { useTenantSupabase } from '@/contexts/supabase-context'

export interface RecentEdit {
  id: string
  type: 'page' | 'post'
  title: string
  slug: string
  updated_at: string
}

export function useRecentEdits() {
  const supabase = useTenantSupabase()

  return useQuery<RecentEdit[]>({
    queryKey: ['recent-edits'],
    queryFn: async () => {
      const [pagesRes, postsRes] = await Promise.all([
        supabase.from('pages').select('id, title, slug, updated_at').order('updated_at', { ascending: false }).limit(5),
        supabase.from('posts').select('id, title, slug, updated_at').order('updated_at', { ascending: false }).limit(5),
      ])

      if (pagesRes.error) throw pagesRes.error
      if (postsRes.error) throw postsRes.error

      const pages: RecentEdit[] = (pagesRes.data ?? []).map((p) => ({ ...p, type: 'page' as const }))
      const posts: RecentEdit[] = (postsRes.data ?? []).map((p) => ({ ...p, type: 'post' as const }))

      return [...pages, ...posts]
        .sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())
        .slice(0, 5)
    },
  })
}
