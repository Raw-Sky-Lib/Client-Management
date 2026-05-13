import { useQuery } from '@tanstack/react-query'
import { useTenantSupabase } from '@/contexts/supabase-context'
import type { Page } from '@/types'

export function usePage(slug: string) {
  const supabase = useTenantSupabase()
  return useQuery<Page | null>({
    queryKey: ['page', slug],
    queryFn: async () => {
      const { data, error } = await supabase
        .from('pages')
        .select('*')
        .eq('slug', slug)
        .single()
      if (error) {
        if (error.code === 'PGRST116') return null
        throw error
      }
      return data
    },
    enabled: !!slug,
  })
}
