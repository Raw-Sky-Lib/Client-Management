import { useMutation } from '@tanstack/react-query'
import axios from 'axios'
import api from '@/lib/axios'
import type { FieldChange, GenerateRequest } from '@/types'

export type RateLimitType = 'minute' | 'hour' | 'budget'

export interface RateLimitError {
  type: RateLimitType
  message: string
}

function parseRateLimit(message: string): RateLimitError {
  if (message.includes('too quickly')) {
    return { type: 'minute', message }
  }
  if (message.includes('Hourly limit')) {
    return { type: 'hour', message }
  }
  return { type: 'budget', message }
}

export function useAssistant() {
  return useMutation<FieldChange[], RateLimitError | Error, GenerateRequest>({
    mutationFn: async (req: GenerateRequest) => {
      try {
        const { data } = await api.post<{ changes: FieldChange[] }>('/api/assistant/generate', req)
        return data.changes
      } catch (err) {
        if (axios.isAxiosError(err) && err.response?.status === 429) {
          const msg: string = err.response.data?.error ?? 'Rate limit reached.'
          throw parseRateLimit(msg)
        }
        if (axios.isAxiosError(err)) {
          const msg: string = err.response?.data?.error ?? 'The assistant is temporarily unavailable.'
          throw new Error(msg)
        }
        throw new Error('The assistant is temporarily unavailable.')
      }
    },
  })
}
