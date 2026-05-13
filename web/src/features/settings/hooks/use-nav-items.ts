import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTenantSupabase } from '@/contexts/supabase-context'
import type { NavItem } from '@/types'

export function useNavItems() {
  const supabase = useTenantSupabase()
  return useQuery<NavItem[]>({
    queryKey: ['nav-items'],
    queryFn: async () => {
      const { data, error } = await supabase
        .from('nav_items')
        .select('*')
        .order('order', { ascending: true })
      if (error) throw error
      return data ?? []
    },
  })
}

// Full replace: delete all existing rows, insert the new ordered list.
// Order is derived from array index.
export function useUpdateNavItems() {
  const supabase = useTenantSupabase()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (items: Omit<NavItem, 'id' | 'order'>[]) => {
      const { error: delError } = await supabase.from('nav_items').delete().neq('id', '00000000-0000-0000-0000-000000000000')
      if (delError) throw delError

      if (items.length > 0) {
        const rows = items.map((item, i) => ({ ...item, order: i }))
        const { error: insError } = await supabase.from('nav_items').insert(rows)
        if (insError) throw insError
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nav-items'] })
    },
  })
}
