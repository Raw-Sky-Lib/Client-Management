import { useAuth } from '@/contexts/auth-context'
import { QuickActions } from './components/quick-actions'
import { FormSubmissionsPreview } from './components/form-submissions-preview'
import { RecentEdits } from './components/recent-edits'

function greeting(): string {
  const h = new Date().getHours()
  if (h < 12) return 'Good morning'
  if (h < 17) return 'Good afternoon'
  return 'Good evening'
}

export function DashboardPage() {
  const { user } = useAuth()
  const name = user?.email?.split('@')[0] ?? ''

  return (
    <div className="p-6 md:p-8 max-w-4xl flex flex-col gap-8">
      <div>
        <h1 className="font-mono text-sm font-bold uppercase tracking-widest text-ink">
          {greeting()}{name ? `, ${name}` : ''}
        </h1>
        {user?.site_url && (
          <p className="font-mono text-[0.65rem] text-ink/35 mt-1">
            Managing <span className="text-ink/60">{new URL(user.site_url).hostname}</span>
          </p>
        )}
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <QuickActions />
        <FormSubmissionsPreview />
      </div>

      <RecentEdits />
    </div>
  )
}
