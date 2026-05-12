import axios, { isAxiosError } from 'axios'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL as string,
  withCredentials: true,
})

// Attach CSRF token from cookie to every mutating request.
// The cookie is set by GET /api/auth/csrf (called at app startup in AuthProvider).
api.interceptors.request.use((config) => {
  const match = document.cookie.split('; ').find((c) => c.startsWith('csrf_token='))
  if (match) config.headers['X-CSRF-Token'] = match.split('=')[1]
  return config
})

// Single in-flight refresh gate — concurrent 401s share one refresh call instead of racing.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let refreshing: Promise<any> | null = null

api.interceptors.response.use(
  (res) => res,
  async (err) => {
    // Use type assertion so we can tag the config object to prevent infinite retry loops.
    const config = err.config as typeof err.config & { _refreshed?: boolean; _csrfed?: boolean }

    // 401 — silently refresh the access token, then replay the original request once.
    // Mark the refresh request itself with _refreshed so it does not re-enter this branch
    // if the refresh endpoint also returns 401 (expired/missing refresh token).
    if (isAxiosError(err) && err.response?.status === 401 && !config._refreshed) {
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

    // 403 — stale or missing CSRF cookie. Re-bootstrap the token and replay once.
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    if (isAxiosError(err) && err.response?.status === 403 && !config._csrfed) {
      config._csrfed = true
      try {
        // Mark the CSRF fetch itself so a 403 on it doesn't loop.
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        await api.get('/api/auth/csrf', { _csrfed: true } as any)
        return api(config)
      } catch {
        return Promise.reject(err)
      }
    }

    return Promise.reject(err)
  },
)

export default api
