import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { FileObject } from '@supabase/storage-js'
import axios from 'axios'
import { useTenantSupabase } from '@/contexts/supabase-context'
import api from '@/lib/axios'

export const IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp', 'image/gif', 'image/svg+xml']
export const VIDEO_TYPES = ['video/mp4', 'video/webm', 'video/ogg', 'video/quicktime']
export const ACCEPTED_TYPES = [...IMAGE_TYPES, ...VIDEO_TYPES]
export const MAX_IMAGE_BYTES = 5 * 1024 * 1024
export const MAX_VIDEO_BYTES = 10 * 1024 * 1024

export type StorageItem = FileObject & { isFolder: boolean }

// Extracts a user-readable message from an unknown error, including axios API errors.
export function extractErrorMessage(err: unknown): string {
  if (axios.isAxiosError(err)) {
    const msg = err.response?.data?.error
    if (typeof msg === 'string' && msg) return msg
    if (err.code === 'ERR_NETWORK') return 'Network error — check your connection.'
    if (err.code === 'ECONNABORTED') return 'Request timed out.'
  }
  if (err instanceof Error) return err.message
  return 'Upload failed.'
}

// Validates a file client-side before attempting upload. Returns an error string or null.
export function validateFile(file: File): string | null {
  if (file.size === 0) return 'File is empty.'
  if (!ACCEPTED_TYPES.includes(file.type)) {
    return 'Unsupported type. Accepted: JPEG, PNG, WebP, GIF, SVG, MP4, WebM.'
  }
  const isVideo = VIDEO_TYPES.includes(file.type)
  const maxBytes = isVideo ? MAX_VIDEO_BYTES : MAX_IMAGE_BYTES
  const limitMB = maxBytes / 1024 / 1024
  if (file.size > maxBytes) return `Exceeds ${limitMB} MB limit.`
  return null
}

export function useInitBucket() {
  return useQuery<{ bucket: string }>({
    queryKey: ['media', 'init-bucket'],
    queryFn: () =>
      api.post<{ bucket: string }>('/api/media/init-bucket').then((r) => r.data),
    staleTime: Infinity,
    retry: 2,
  })
}

// Lists files/folders via the backend (service role key) — bypasses Storage RLS.
export function useStorageFiles(bucket: string, path: string) {
  return useQuery<StorageItem[]>({
    queryKey: ['storage', bucket, path],
    queryFn: async () => {
      const { data } = await api.get<FileObject[]>('/api/media/files', {
        params: { bucket, path },
      })
      return (data ?? []).map((item) => ({
        ...item,
        isFolder: item.id === null,
      }))
    },
    enabled: !!bucket,
  })
}

// Uploads via the backend (service role key).
// Path is passed per-call so the destination can be chosen at upload time.
export function useUploadFile(bucket: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({
      file,
      folderPath,
      onProgress,
    }: {
      file: File
      folderPath: string
      onProgress?: (percent: number) => void
    }) => {
      const validationError = validateFile(file)
      if (validationError) throw new Error(validationError)

      // Sanitize filename: replace whitespace and special chars with underscores
      const safeName = file.name.replace(/\s+/g, '_').replace(/[^a-zA-Z0-9._-]/g, '_')

      const form = new FormData()
      form.append('file', file, safeName)
      form.append('bucket', bucket)
      form.append('path', folderPath)

      try {
        const { data } = await api.post<{ url: string; path: string }>('/api/media/upload', form, {
          onUploadProgress: (e) => {
            if (e.total) onProgress?.(Math.round((e.loaded / e.total) * 100))
          },
        })
        return data
      } catch (err) {
        throw new Error(extractErrorMessage(err))
      }
    },
    onSuccess: (_, { folderPath }) => {
      queryClient.invalidateQueries({ queryKey: ['storage', bucket, folderPath] })
      const parent = folderPath.includes('/')
        ? folderPath.split('/').slice(0, -1).join('/')
        : ''
      if (parent !== folderPath) {
        queryClient.invalidateQueries({ queryKey: ['storage', bucket, parent] })
      }
    },
  })
}

// Deletes via the backend (service role key) — bypasses Storage RLS.
export function useDeleteFile(bucket: string, folderPath: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (path: string) => {
      try {
        await api.delete('/api/media/file', { data: { bucket, path } })
      } catch (err) {
        throw new Error(extractErrorMessage(err))
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['storage', bucket, folderPath] })
    },
  })
}

export function getPublicUrl(
  supabase: ReturnType<typeof useTenantSupabase>,
  bucket: string,
  path: string,
): string {
  return supabase.storage.from(bucket).getPublicUrl(path).data.publicUrl
}
