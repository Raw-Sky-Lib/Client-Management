import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router'
import { isAxiosError } from 'axios'
import { AgencyBadge } from '@/components/ui/agency-badge'
import api from '@/lib/axios'

export function AuthCallbackPage() {
  const navigate = useNavigate()
  const [error, setError]   = useState<string | null>(null)

  useEffect(() => {
    // Supabase delivers tokens in the URL fragment, not query params.
    // e.g. /auth/callback#access_token=...&type=magiclink
    const params           = new URLSearchParams(window.location.hash.slice(1))
    const accessToken      = params.get('access_token')
    const errorDescription = params.get('error_description')

    if (errorDescription) {
      setError(errorDescription)
      return
    }
    if (!accessToken) {
      setError('No access token found. The link may be malformed.')
      return
    }

    api
      .post('/api/auth/exchange', { access_token: accessToken })
      .then(() => navigate('/dashboard', { replace: true }))
      .catch((err) => {
        if (isAxiosError(err)) {
          setError(err.response?.data?.error ?? 'Sign-in failed. Please try again.')
        } else {
          setError('Something went wrong. Please try again.')
        }
      })
  }, [navigate])

  if (error) {
    return (
      <div className="min-h-svh bg-cream flex flex-col items-center justify-center p-8 gap-8">
        <div className="flex items-center gap-2.5 font-sans font-extrabold text-2xl tracking-tight text-ink">
          <div className="w-3 h-3 rounded-full bg-brand-red border-2 border-ink shrink-0" />
          Client Portal
        </div>
        <div className="w-full max-w-md border-2 border-ink rounded-card bg-white px-8 py-7 text-center"
          style={{ boxShadow: '8px 8px 0 #1C1C1A' }}>
          <div className="w-8 h-8 rounded-full bg-brand-red border-2 border-ink flex items-center justify-center mx-auto mb-4">
            <span className="text-white font-mono font-bold text-sm leading-none">✕</span>
          </div>
          <h2 className="font-sans font-extrabold text-xl text-ink mb-2">Sign-in failed.</h2>
          <p className="font-mono text-sm text-ink opacity-60 mb-6">{error}</p>
          <a
            href="/login"
            className="font-mono text-sm text-ink underline opacity-60 hover:opacity-100 transition"
          >
            ← Back to sign in
          </a>
        </div>
        <AgencyBadge />
      </div>
    )
  }

  return (
    <div className="min-h-svh bg-cream flex flex-col items-center justify-center p-8 gap-6">
      <div className="flex items-center gap-2.5 font-sans font-extrabold text-2xl tracking-tight text-ink">
        <div className="w-3 h-3 rounded-full bg-brand-red border-2 border-ink shrink-0" />
        Client Portal
      </div>
      <div className="flex flex-col items-center gap-3">
        <div className="w-6 h-6 border-2 border-ink/30 border-t-ink rounded-full animate-spin" />
        <p className="font-mono text-sm text-ink opacity-60">Signing you in…</p>
      </div>
      <AgencyBadge />
    </div>
  )
}
