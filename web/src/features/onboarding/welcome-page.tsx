import { useNavigate } from 'react-router'
import { FileText, PenLine, Image, Inbox, Settings2, Sparkles } from 'lucide-react'
import { OnboardingLayout, type OnboardingStep } from '@/components/layout/onboarding-layout'
import { HardShadowCard } from '@/components/ui/hard-shadow-card'

const STEPS: OnboardingStep[] = [
  { label: 'Invite Sent',     sublabel: 'Link delivered',    status: 'done'   },
  { label: 'Email Confirmed', sublabel: 'Identity verified', status: 'done'   },
  { label: 'Access Granted',  sublabel: 'Workspace ready',   status: 'active' },
]

const FEATURES = [
  { icon: FileText,  label: 'Pages',             desc: 'Edit website sections and content'   },
  { icon: PenLine,   label: 'Blog',              desc: 'Publish and manage blog posts'        },
  { icon: Image,     label: 'Media',             desc: 'Upload and organise images and files' },
  { icon: Inbox,     label: 'Forms',             desc: 'View contact form submissions'        },
  { icon: Settings2, label: 'Settings',          desc: 'Site name, SEO and navigation'        },
  { icon: Sparkles,  label: 'Content Assistant', desc: 'AI-powered copy suggestions'          },
]

export function WelcomePage() {
  const navigate = useNavigate()

  return (
    <OnboardingLayout steps={STEPS}>
      <HardShadowCard className="flex flex-col flex-1">

        <div className="px-10 pt-8 pb-6 border-b-2 border-ink">
          <h2 className="font-sans font-extrabold text-[2rem] leading-tight tracking-tight text-ink">
            Your portal is ready.
          </h2>
          <p className="font-mono text-sm text-ink opacity-60 mt-1">
            Head to your dashboard to start managing your site.
          </p>
        </div>

        <div className="flex flex-col flex-1 px-10 py-8 gap-8 overflow-y-auto">

          <div className="flex flex-col gap-2">
            <p className="font-mono text-xs uppercase tracking-widest text-ink opacity-50">
              What you can manage
            </p>
            <div className="grid grid-cols-2 gap-3">
              {FEATURES.map(({ icon: Icon, label, desc }) => (
                <div
                  key={label}
                  className="flex items-start gap-3 border-2 border-ink/15 rounded-lg px-4 py-3"
                >
                  <Icon className="w-4 h-4 shrink-0 text-ink mt-0.5" />
                  <div>
                    <p className="font-sans font-bold text-sm text-ink leading-tight">{label}</p>
                    <p className="font-mono text-[0.7rem] text-ink opacity-50 mt-0.5">{desc}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="flex-1" />

          <div className="border-t-2 border-dashed border-ink pt-6">
            <button
              type="button"
              onClick={() => navigate('/dashboard')}
              className="btn-ink-shadow w-full flex items-center justify-center gap-3 bg-forest text-white font-sans font-extrabold text-lg uppercase tracking-wider py-4 px-8 rounded-lg border-2 border-ink"
            >
              Go to my dashboard
              <span className="text-xl leading-none">⇲</span>
            </button>
          </div>

        </div>
      </HardShadowCard>
    </OnboardingLayout>
  )
}
