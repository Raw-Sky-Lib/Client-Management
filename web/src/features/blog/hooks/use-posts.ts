import { useQuery } from '@tanstack/react-query'
import { useTenantSupabase } from '@/contexts/supabase-context'
import type { Post } from '@/types'

type PostListItem = Pick<Post, 'id' | 'slug' | 'title' | 'is_published' | 'published_at' | 'updated_at'>

export function usePosts() {
  const supabase = useTenantSupabase()
  return useQuery<PostListItem[]>({
    queryKey: ['posts'],
    queryFn: async () => {
      const { data, error } = await supabase
        .from('posts')
        .select('id, slug, title, is_published, published_at, updated_at')
        .order('created_at', { ascending: false })
      if (error) throw error
      return data ?? []
    },
  })
}
