import { useState } from 'react'
import { OnboardingLayout, type OnboardingStep } from '@/components/layout/onboarding-layout'
import { ConnectForm } from './connect-form'
import { CheckEmailScreen } from './check-email-screen'

type View = 'form' | 'check-email'

const STEPS_FORM: OnboardingStep[] = [
  { label: 'Enter Code',     sublabel: 'Enter your access code',  status: 'active'  },
  { label: 'Verify Email',   sublabel: 'We will send you a link', status: 'pending' },
  { label: 'Access Granted', sublabel: 'Your workspace is ready', status: 'pending' },
]

const STEPS_CHECK_EMAIL: OnboardingStep[] = [
  { label: 'Enter Code',     sublabel: 'Code verified',           status: 'done'    },
  { label: 'Verify Email',   sublabel: 'Check your inbox',        status: 'active'  },
  { label: 'Access Granted', sublabel: 'Your workspace is ready', status: 'pending' },
]

export function ConnectPage() {
  const [view, setView]   = useState<View>('form')
  const [email, setEmail] = useState('')

  function handleSuccess(submittedEmail: string) {
    setEmail(submittedEmail)
    setView('check-email')
  }

  return (
    <OnboardingLayout steps={view === 'check-email' ? STEPS_CHECK_EMAIL : STEPS_FORM}>
      {view === 'check-email'
        ? <CheckEmailScreen email={email} />
        : <ConnectForm onSuccess={handleSuccess} />
      }
    </OnboardingLayout>
  )
}
