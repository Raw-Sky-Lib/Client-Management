import type { HeroSection } from '@/types'
import type { SectionEditorProps } from './section-editor'
import { Field, inputClass, textareaClass } from './editor-primitives'

export function HeroEditor({ value, onChange }: SectionEditorProps) {
  const hero = value as Partial<HeroSection>

  function set(field: keyof HeroSection, val: string) {
    onChange({ ...value, [field]: val })
  }

  return (
    <div className="flex flex-col gap-5">
      <Field label="Headline">
        <input
          type="text"
          value={hero.headline ?? ''}
          onChange={e => set('headline', e.target.value)}
          placeholder="Your compelling headline"
          className={inputClass}
        />
      </Field>

      <Field label="Subheadline">
        <textarea
          rows={3}
          value={hero.subheadline ?? ''}
          onChange={e => set('subheadline', e.target.value)}
          placeholder="Supporting text beneath the headline"
          className={textareaClass}
        />
      </Field>

      <div className="grid grid-cols-2 gap-4">
        <Field label="CTA Label">
          <input
            type="text"
            value={hero.cta_label ?? ''}
            onChange={e => set('cta_label', e.target.value)}
            placeholder="Get started"
            className={inputClass}
          />
        </Field>
        <Field label="CTA URL">
          <input
            type="text"
            value={hero.cta_url ?? ''}
            onChange={e => set('cta_url', e.target.value)}
            placeholder="/contact"
            className={inputClass}
          />
        </Field>
      </div>
    </div>
  )
}
