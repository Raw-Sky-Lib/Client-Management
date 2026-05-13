import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTenantSupabase } from '@/contexts/supabase-context'
import type { FormSubmission } from '@/types'

export function useSubmissions() {
  const supabase = useTenantSupabase()
  return useQuery<FormSubmission[]>({
    queryKey: ['submissions'],
    queryFn: async () => {
      const { data, error } = await supabase
        .from('form_submissions')
        .select('*')
        .order('submitted_at', { ascending: false })
      if (error) throw error
      return data ?? []
    },
  })
}

export function useMarkRead() {
  const supabase = useTenantSupabase()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await supabase
        .from('form_submissions')
        .update({ is_read: true })
        .eq('id', id)
      if (error) throw error
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['submissions'] })
    },
  })
}
