import { Plus, Trash2 } from 'lucide-react'
import type { WhyUsSection, WhyUsItem } from '@/types'
import type { SectionEditorProps } from './section-editor'
import { Field, inputClass, textareaClass } from './editor-primitives'

const EMPTY_ITEM: WhyUsItem = { icon: '', title: '', description: '' }

export function WhyUsEditor({ value, onChange }: SectionEditorProps) {
  const section = value as Partial<WhyUsSection>
  const items: WhyUsItem[] = section.items ?? []

  function updateItem(idx: number, field: keyof WhyUsItem, val: string) {
    onChange({ ...value, items: items.map((item, i) => i === idx ? { ...item, [field]: val } : item) })
  }

  return (
    <div className="flex flex-col gap-6">
      <Field label="Heading">
        <input type="text" value={section.title ?? ''} onChange={e => onChange({ ...value, title: e.target.value })} className={inputClass} />
      </Field>
      <Field label="Subheading">
        <textarea rows={2} value={section.subtitle ?? ''} onChange={e => onChange({ ...value, subtitle: e.target.value })} className={textareaClass} />
      </Field>

      {items.map((item, idx) => (
        <div key={idx} className="border-2 border-ink/10 rounded-lg p-4 flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <span className="font-mono text-xs text-ink/40 uppercase tracking-widest">Item {idx + 1}</span>
            <button type="button" onClick={() => onChange({ ...value, items: items.filter((_, i) => i !== idx) })} className="text-brand-red opacity-50 hover:opacity-100 transition">
              <Trash2 size={14} />
            </button>
          </div>
          <Field label="Icon"><input type="text" value={item.icon ?? ''} onChange={e => updateItem(idx, 'icon', e.target.value)} placeholder="◎" className={inputClass} /></Field>
          <Field label="Title"><input type="text" value={item.title} onChange={e => updateItem(idx, 'title', e.target.value)} className={inputClass} /></Field>
          <Field label="Description"><textarea rows={2} value={item.description} onChange={e => updateItem(idx, 'description', e.target.value)} className={textareaClass} /></Field>
        </div>
      ))}

      <button type="button" onClick={() => onChange({ ...value, items: [...items, { ...EMPTY_ITEM }] })} className="flex items-center gap-2 font-mono text-xs text-ink/50 hover:text-ink transition border-2 border-dashed border-ink/20 hover:border-ink/40 rounded-lg px-4 py-3 w-full justify-center">
        <Plus size={14} />Add reason
      </button>
    </div>
  )
}
