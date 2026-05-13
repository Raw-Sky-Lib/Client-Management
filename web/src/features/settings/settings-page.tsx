import * as Tabs from '@radix-ui/react-tabs'
import { GeneralSettings } from './components/general-settings'
import { SeoSettings } from './components/seo-settings'
import { SocialSettings } from './components/social-settings'
import { NavEditor } from './components/nav-editor'

const TABS = [
  { id: 'general', label: 'General',  component: <GeneralSettings /> },
  { id: 'seo',     label: 'SEO',      component: <SeoSettings /> },
  { id: 'social',  label: 'Social',   component: <SocialSettings /> },
  { id: 'nav',     label: 'Nav',      component: <NavEditor /> },
] as const

export function SettingsPage() {
  return (
    <div className="p-6 md:p-8 max-w-xl">
      <h1 className="font-mono text-sm font-bold uppercase tracking-widest text-ink mb-6">Settings</h1>

      <Tabs.Root defaultValue="general">
        <Tabs.List className="flex gap-0 border-b-2 border-ink/10 mb-6">
          {TABS.map((tab) => (
            <Tabs.Trigger
              key={tab.id}
              value={tab.id}
              className="relative px-4 py-2.5 font-mono text-[0.65rem] uppercase tracking-widest text-ink/40 hover:text-ink transition data-[state=active]:text-ink after:absolute after:bottom-[-2px] after:left-0 after:right-0 after:h-[2px] after:bg-ink after:opacity-0 data-[state=active]:after:opacity-100 after:transition"
            >
              {tab.label}
            </Tabs.Trigger>
          ))}
        </Tabs.List>

        {TABS.map((tab) => (
          <Tabs.Content key={tab.id} value={tab.id} className="focus:outline-none">
            {tab.component}
          </Tabs.Content>
        ))}
      </Tabs.Root>
    </div>
  )
}
