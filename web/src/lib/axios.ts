import axios from 'axios'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL as string,
  withCredentials: true,
})

api.interceptors.request.use((config) => {
  const match = document.cookie
    .split('; ')
    .find((c) => c.startsWith('csrf_token='))
  if (match) {
    config.headers['X-CSRF-Token'] = match.split('=')[1]
  }
  return config
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      window.location.href = '/connect'
    }
    return Promise.reject(err)
  },
)

export default api
