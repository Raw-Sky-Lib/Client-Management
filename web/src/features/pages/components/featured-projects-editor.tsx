import type { FeaturedProjectsSection } from '@/types'
import type { SectionEditorProps } from './section-editor'
import { Field, inputClass, textareaClass } from './editor-primitives'

export function FeaturedProjectsEditor({ value, onChange }: SectionEditorProps) {
  const section = value as Partial<FeaturedProjectsSection>

  return (
    <div className="flex flex-col gap-5">
      <p className="font-mono text-[0.6rem] text-ink/30 leading-relaxed">
        Projects are pulled automatically from the Work section. Edit which ones are featured there.
      </p>
      <Field label="Heading">
        <input type="text" value={section.title ?? ''} onChange={e => onChange({ ...value, title: e.target.value })} className={inputClass} />
      </Field>
      <Field label="Subheading">
        <textarea rows={2} value={section.subtitle ?? ''} onChange={e => onChange({ ...value, subtitle: e.target.value })} className={textareaClass} />
      </Field>
      <Field label="Link text">
        <input type="text" value={section.cta_label ?? ''} onChange={e => onChange({ ...value, cta_label: e.target.value })} placeholder="View All Work" className={inputClass} />
      </Field>
      <Field label="Link URL">
        <input type="text" value={section.cta_url ?? ''} onChange={e => onChange({ ...value, cta_url: e.target.value })} placeholder="/work" className={inputClass} />
      </Field>
    </div>
  )
}
