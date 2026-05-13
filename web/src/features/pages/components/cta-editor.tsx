import type { CTASection } from '@/types'
import type { SectionEditorProps } from './section-editor'
import { Field, inputClass, textareaClass } from './editor-primitives'

export function CTAEditor({ value, onChange }: SectionEditorProps) {
  const cta = value as Partial<CTASection>

  function set(field: keyof CTASection, val: string) {
    onChange({ ...value, [field]: val })
  }

  return (
    <div className="flex flex-col gap-5">
      <Field label="Headline">
        <input
          type="text"
          value={cta.headline ?? ''}
          onChange={e => set('headline', e.target.value)}
          placeholder="Ready to get started?"
          className={inputClass}
        />
      </Field>

      <Field label="Subheadline">
        <textarea
          rows={2}
          value={cta.subheadline ?? ''}
          onChange={e => set('subheadline', e.target.value)}
          placeholder="Supporting text for the CTA"
          className={textareaClass}
        />
      </Field>

      <div className="grid grid-cols-2 gap-4">
        <Field label="Button Label">
          <input
            type="text"
            value={cta.button_label ?? ''}
            onChange={e => set('button_label', e.target.value)}
            placeholder="Get in touch"
            className={inputClass}
          />
        </Field>
        <Field label="Button URL">
          <input
            type="text"
            value={cta.button_url ?? ''}
            onChange={e => set('button_url', e.target.value)}
            placeholder="/contact"
            className={inputClass}
          />
        </Field>
      </div>
    </div>
  )
}
