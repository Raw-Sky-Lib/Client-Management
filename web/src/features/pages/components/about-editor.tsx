import type { AboutSection } from '@/types'
import type { SectionEditorProps } from './section-editor'
import { Field, textareaClass } from './editor-primitives'
import { ImagePickerField } from './image-picker-field'

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

      <ImagePickerField
        label="Section Image"
        value={about.image_url ?? ''}
        onChange={url => onChange({ ...value, image_url: url })}
      />
    </div>
  )
}
