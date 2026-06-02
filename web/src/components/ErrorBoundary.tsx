import { Component, type ReactNode } from 'react'

interface Props {
  children: ReactNode
  fallback?: ReactNode
}

interface State {
  hasError: boolean
  message: string
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, message: '' }

  static getDerivedStateFromError(error: unknown): State {
    const message = error instanceof Error ? error.message : 'An unexpected error occurred.'
    return { hasError: true, message }
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback
      return (
        <div className="min-h-svh bg-cream flex flex-col items-center justify-center gap-4 p-8">
          <div className="w-full max-w-md border-2 border-ink rounded-lg bg-white px-8 py-7 text-center"
            style={{ boxShadow: '6px 6px 0 #1C1C1A' }}>
            <div className="w-8 h-8 rounded-full bg-brand-red border-2 border-ink flex items-center justify-center mx-auto mb-4">
              <span className="text-white font-mono font-bold text-sm leading-none">!</span>
            </div>
            <h2 className="font-sans font-extrabold text-xl text-ink mb-2">Something went wrong</h2>
            <p className="font-mono text-sm text-ink/60 mb-6">{this.state.message}</p>
            <button
              type="button"
              onClick={() => window.location.reload()}
              className="font-mono text-sm text-ink underline opacity-60 hover:opacity-100 transition"
            >
              Reload page
            </button>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}
