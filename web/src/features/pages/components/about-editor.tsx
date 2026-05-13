import type { AboutSection } from '@/types'
import type { SectionEditorProps } from './section-editor'
import { Field, inputClass, textareaClass } from './editor-primitives'

export function AboutEditor({ value, onChange }: SectionEditorProps) {
  const about = value as Partial<AboutSection>

  return (
    <div className="flex flex-col gap-5">
      <Field label="Body">
        <textarea
          rows={6}
          value={about.body ?? ''}
          onChange={e => onChange({ ...value, body: e.target.value })}
          placeholder="Write your about section content here…"
          className={textareaClass}
        />
      </Field>

      <Field label="Image URL">
        <input
          type="text"
          value={about.image_url ?? ''}
          onChange={e => onChange({ ...value, image_url: e.target.value })}
          placeholder="https://… (optional)"
          className={inputClass}
        />
      </Field>
    </div>
  )
}
