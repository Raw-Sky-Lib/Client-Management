import { useQuery } from '@tanstack/react-query'
import { useTenantSupabase } from '@/contexts/supabase-context'
import type { Page } from '@/types'

type PageListItem = Pick<Page, 'id' | 'slug' | 'title' | 'is_published' | 'updated_at'>

export function usePages() {
  const supabase = useTenantSupabase()
  return useQuery<PageListItem[]>({
    queryKey: ['pages'],
    queryFn: async () => {
      const { data, error } = await supabase
        .from('pages')
        .select('id, slug, title, is_published, updated_at')
        .order('slug')
      if (error) throw error
      return data ?? []
    },
  })
}
