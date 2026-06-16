import type { EmbedSection } from '@/types'
import type { SectionEditorProps } from './section-editor'
import { Field, inputClass } from './editor-primitives'
import { EmbedCodeField } from './embed-code-field'

export function EmbedEditor({ value, onChange }: SectionEditorProps) {
  const section = value as Partial<EmbedSection>

  return (
    <div className="flex flex-col gap-5">
      <EmbedCodeField
        label="Embed Code"
        value={section.embed_code ?? ''}
        onChange={code => onChange({ ...value, embed_code: code })}
      />

      <Field label="Caption">
        <input
          type="text"
          value={section.caption ?? ''}
          onChange={e => onChange({ ...value, caption: e.target.value })}
          placeholder="Optional caption displayed below the embed"
          className={inputClass}
        />
      </Field>
    </div>
  )
}
