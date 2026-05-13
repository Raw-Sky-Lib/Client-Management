import { Link } from 'react-router'
import { FilePlus, PenLine, ImagePlus, Inbox, ArrowRight } from 'lucide-react'

const ACTIONS = [
  { label: 'New Page',        href: '/pages',     icon: FilePlus,  description: 'Add a page to your site' },
  { label: 'New Post',        href: '/blog/new',  icon: PenLine,   description: 'Write a blog post' },
  { label: 'Upload Media',    href: '/media',     icon: ImagePlus, description: 'Add images or files' },
  { label: 'View Submissions',href: '/forms',     icon: Inbox,     description: 'See form responses' },
] as const

export function QuickActions() {
  return (
    <div>
      <h2 className="font-mono text-[0.65rem] uppercase tracking-widest text-ink/40 mb-3">Quick actions</h2>
      <div className="grid grid-cols-2 gap-2">
        {ACTIONS.map(({ label, href, icon: Icon, description }) => (
          <Link
            key={href}
            to={href}
            className="group flex items-start gap-3 border-2 border-ink/10 rounded-xl p-4 hover:border-ink/30 hover:bg-ink/[0.02] transition"
          >
            <div className="mt-0.5 rounded-lg bg-ink/5 p-2 group-hover:bg-ink/10 transition">
              <Icon size={14} className="text-ink/60" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="font-mono text-xs font-bold text-ink">{label}</p>
              <p className="font-mono text-[0.6rem] text-ink/40 mt-0.5 leading-tight">{description}</p>
            </div>
            <ArrowRight size={12} className="text-ink/20 group-hover:text-ink/50 mt-1 transition shrink-0" />
          </Link>
        ))}
      </div>
    </div>
  )
}
