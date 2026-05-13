import { useQuery } from '@tanstack/react-query'
import { useTenantSupabase } from '@/contexts/supabase-context'
import type { Post } from '@/types'

export function usePost(id: string) {
  const supabase = useTenantSupabase()
  return useQuery<Post | null>({
    queryKey: ['post', id],
    queryFn: async () => {
      const { data, error } = await supabase
        .from('posts')
        .select('*')
        .eq('id', id)
        .single()
      if (error) {
        if (error.code === 'PGRST116') return null
        throw error
      }
      return data
    },
    enabled: !!id,
  })
}
