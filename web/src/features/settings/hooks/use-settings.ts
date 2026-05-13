import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTenantSupabase } from '@/contexts/supabase-context'

export function useSetting(key: string) {
  const supabase = useTenantSupabase()
  return useQuery<string>({
    queryKey: ['settings', key],
    queryFn: async () => {
      const { data, error } = await supabase
        .from('site_settings')
        .select('value')
        .eq('key', key)
        .maybeSingle()
      if (error) throw error
      return data?.value ?? ''
    },
  })
}

export function useUpdateSetting() {
  const supabase = useTenantSupabase()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ key, value }: { key: string; value: string }) => {
      const { error } = await supabase
        .from('site_settings')
        .upsert({ key, value, updated_at: new Date().toISOString() }, { onConflict: 'key' })
      if (error) throw error
    },
    onSuccess: (_, { key }) => {
      queryClient.invalidateQueries({ queryKey: ['settings', key] })
    },
  })
}

// Saves multiple settings at once — used by each tab's Save button
export function useUpdateSettings() {
  const supabase = useTenantSupabase()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (entries: { key: string; value: string }[]) => {
      const rows = entries.map((e) => ({
        key: e.key,
        value: e.value,
        updated_at: new Date().toISOString(),
      }))
      const { error } = await supabase
        .from('site_settings')
        .upsert(rows, { onConflict: 'key' })
      if (error) throw error
    },
    onSuccess: (_, entries) => {
      entries.forEach(({ key }) => {
        queryClient.invalidateQueries({ queryKey: ['settings', key] })
      })
    },
  })
}
