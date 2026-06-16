import type { ContactSection } from '@/types'
import type { SectionEditorProps } from './section-editor'
import { Field, inputClass } from './editor-primitives'
import { EmbedCodeField } from './embed-code-field'
import { ImagePickerField } from './image-picker-field'

export function ContactEditor({ value, onChange }: SectionEditorProps) {
  const contact = value as Partial<ContactSection>

  function set(field: keyof ContactSection, val: string) {
    onChange({ ...value, [field]: val })
  }

  return (
    <div className="flex flex-col gap-5">
      <Field label="Title">
        <input
          type="text"
          value={contact.title ?? ''}
          onChange={e => set('title', e.target.value)}
          placeholder="Find Us"
          className={inputClass}
        />
      </Field>

      <ImagePickerField
        label="Photo"
        value={contact.image_url ?? ''}
        onChange={url => set('image_url', url)}
      />

      <Field label="Address">
        <input
          type="text"
          value={contact.address ?? ''}
          onChange={e => set('address', e.target.value)}
          placeholder="123 Main St, City, State"
          className={inputClass}
        />
      </Field>

      <Field label="Email">
        <input
          type="email"
          value={contact.email ?? ''}
          onChange={e => set('email', e.target.value)}
          placeholder="hello@yourbusiness.com"
          className={inputClass}
        />
      </Field>

      <Field label="Phone">
        <input
          type="text"
          value={contact.phone ?? ''}
          onChange={e => set('phone', e.target.value)}
          placeholder="+1 (555) 000-0000"
          className={inputClass}
        />
      </Field>

      <EmbedCodeField
        label="Map Embed"
        value={contact.map_embed ?? ''}
        onChange={code => set('map_embed', code)}
        placeholder='<iframe src="https://www.google.com/maps/embed?..." ...></iframe>'
      />
    </div>
  )
}
