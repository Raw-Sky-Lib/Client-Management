import axios, { isAxiosError } from 'axios'
import { toast } from 'sonner'

// ─── CSRF token — stored in memory, never localStorage ───────────────────────
// Set from the response body of GET /api/auth/csrf, which also sets a cookie.
// Storing it in memory prevents XSS from reading the token.
let csrfToken: string | null = null

export function setCSRFToken(token: string) {
  csrfToken = token
}

export function getCSRFToken(): string | null {
  return csrfToken
}

// ─── Axios instance ───────────────────────────────────────────────────────────
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL as string,
  withCredentials: true,
})

// ─── Request interceptor: attach in-memory CSRF to mutations ─────────────────
const MUTATION_METHODS = new Set(['post', 'put', 'patch', 'delete'])

api.interceptors.request.use((config) => {
  const method = config.method?.toLowerCase() ?? ''
  if (MUTATION_METHODS.has(method) && csrfToken) {
    config.headers['X-CSRF-Token'] = csrfToken
  }
  return config
})

// ─── Single in-flight refresh gate ───────────────────────────────────────────
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let refreshing: Promise<any> | null = null

// ─── Response interceptor ────────────────────────────────────────────────────
api.interceptors.response.use(
  (res) => res,
  async (err) => {
    const config = err.config as typeof err.config & { _refreshed?: boolean; _csrfed?: boolean }

    // ── No response at all (network down, server unreachable) ────────────────
    if (!err.response) {
      toast.error('Network error — check your connection and try again.')
      return Promise.reject(err)
    }

    const status: number = err.response.status

    // ── 401: silently refresh access token, then replay once ─────────────────
    if (isAxiosError(err) && status === 401 && !config._refreshed) {
      config._refreshed = true
      try {
        if (!refreshing) {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          refreshing = api.post('/api/auth/refresh', undefined, { _refreshed: true } as any)
            .finally(() => { refreshing = null })
        }
        await refreshing
        return api(config)
      } catch {
        refreshing = null
        return Promise.reject(err)
      }
    }

    // ── 403: stale or missing CSRF — re-bootstrap token and replay once ───────
    if (isAxiosError(err) && status === 403 && !config._csrfed) {
      config._csrfed = true
      try {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const res = await api.get<{ csrf_token: string }>('/api/auth/csrf', { _csrfed: true } as any)
        setCSRFToken(res.data.csrf_token)
        return api(config)
      } catch {
        return Promise.reject(err)
      }
    }

    // ── 429: rate limited ────────────────────────────────────────────────────
    if (status === 429) {
      const msg = err.response.data?.error as string | undefined
      toast.error(msg || 'Too many requests — please wait a moment.')
    }

    // ── 503: external dependency down (Supabase project unreachable) ─────────
    if (status === 503) {
      const msg = err.response.data?.error as string | undefined
      toast.error(msg || 'Service temporarily unavailable. Try the magic link sign-in.')
    }

    // ── 500: unexpected server error ─────────────────────────────────────────
    if (status >= 500 && status !== 503) {
      const msg = err.response.data?.error as string | undefined
      // Only toast generic 500s — specific handlers show inline errors.
      if (!msg || msg === 'something went wrong, please try again') {
        toast.error('Something went wrong on the server. Please try again.')
      }
    }

    return Promise.reject(err)
  },
)

export default api
