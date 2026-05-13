import type { ComponentType } from 'react'
import { HeroEditor } from './hero-editor'
import { FeaturesEditor } from './features-editor'
import { AboutEditor } from './about-editor'
import { TestimonialsEditor } from './testimonials-editor'
import { CTAEditor } from './cta-editor'
import { Field, inputClass, textareaClass } from './editor-primitives'

export interface SectionEditorProps {
  sectionKey: string
  value: Record<string, unknown>
  onChange: (updated: Record<string, unknown>) => void
}

const KNOWN_EDITORS: Record<string, ComponentType<SectionEditorProps>> = {
  hero:         HeroEditor,
  features:     FeaturesEditor,
  about:        AboutEditor,
  testimonials: TestimonialsEditor,
  cta:          CTAEditor,
}

function formatFieldLabel(key: string): string {
  return key.split(/[_-]/).map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')
}

function GenericEditor({ value, onChange }: SectionEditorProps) {
  return (
    <div className="flex flex-col gap-5">
      {Object.entries(value).map(([key, val]) => (
        <Field key={key} label={formatFieldLabel(key)}>
          {typeof val === 'object' && val !== null ? (
            <textarea
              rows={6}
              defaultValue={JSON.stringify(val, null, 2)}
              onBlur={e => {
                try { onChange({ ...value, [key]: JSON.parse(e.target.value) }) } catch { /* invalid JSON */ }
              }}
              className={textareaClass + ' font-mono text-xs'}
            />
          ) : (
            <input
              type="text"
              value={String(val ?? '')}
              onChange={e => onChange({ ...value, [key]: e.target.value })}
              className={inputClass}
            />
          )}
        </Field>
      ))}

      {Object.keys(value).length === 0 && (
        <p className="font-mono text-xs text-ink/40 text-center py-4">
          This section has no fields.
        </p>
      )}
    </div>
  )
}

export function SectionEditor(props: SectionEditorProps) {
  const Editor = KNOWN_EDITORS[props.sectionKey.toLowerCase()] ?? GenericEditor
  return <Editor {...props} />
}
